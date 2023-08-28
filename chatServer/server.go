package chatServer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dawid993/goChat/db"
	"github.com/dawid993/goChat/model"
	"github.com/gorilla/websocket"
)

const (
	ServerCertPath = "cert/serv/server-cert.pem"
	ServerKeyPath  = "cert/serv/server-key.pem"
	CACertPath     = "cert/ca/ca-cert.pem"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var checkOriginFn = func(r *http.Request) bool {
	return true
}

type Server struct {
	addr string
	port int
	db   *db.DatabaseHandler
}

func (s *Server) Run(r *model.Room) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	s.setupAndRunRoom(ctx, r)

	httpServer := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", s.addr, s.port),
		TLSConfig: s.getTlsConfig(),
		Handler:   s.newRouter(ctx, r),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)

	go func() {
		fmt.Println("Server is running!")
		if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-stop

	cancelCtx()

	fmt.Println("Server is shutting down...")
	fmt.Println(r.Clients)
	for client, _ := range r.Clients {
		fmt.Println("a")
		<-client.Closed
		fmt.Println("b")

	}

	fmt.Println("All connection closed")
	// Create a deadline to wait for.
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout (or until all connections are closed).
	httpServer.Shutdown(ctxShutdown)
	fmt.Println("Server stopped")
}

func (s *Server) newRouter(ctx context.Context, r *model.Room) *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		s.registerClient(ctx, r, w, req)
	})

	return router
}

func (*Server) setupAndRunRoom(ctx context.Context, r *model.Room) {
	db := db.NewDatabaseHandler()
	r.Db = db

	go r.Run(ctx)
}

func (s *Server) registerClient(ctx context.Context, r *model.Room, w http.ResponseWriter, req *http.Request) {
	upgrader.CheckOrigin = checkOriginFn
	socket, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		log.Fatal(err)
	}

	client := s.setupClient(socket, r, ctx)

	messages, err := r.Db.GetAllMessages(ctx)

	for _, msg := range messages {
		client.Message <- model.Message{From: msg.From, Content: msg.Content}
	}
}

func (*Server) setupClient(socket *websocket.Conn, r *model.Room, ctx context.Context) *model.Client {
	client := &model.Client{
		Conn:    socket,
		Message: make(chan model.Message),
		Room:    r,
		Closed:  make(chan bool),
	}

	r.Register <- client

	go client.Read(ctx)
	go client.Write(ctx)

	return client
}

func (s *Server) getTlsConfig() *tls.Config {
	serverCert, err := tls.LoadX509KeyPair(ServerCertPath, ServerKeyPath)
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}

	caCert, err := ioutil.ReadFile(CACertPath)
	if err != nil {
		log.Fatalf("server: read ca: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{serverCert},
	}
}

func NewServer(addr string, port int) *Server {
	return &Server{
		addr: addr,
		port: port,
	}
}
