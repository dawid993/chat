package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var (
	interrupt = make(chan os.Signal, 1)
)

func main() {
	signal.Notify(interrupt, os.Interrupt)

	cert, err := tls.LoadX509KeyPair("cert/client/zst-cert.pem", "cert/client/zst-key.pem")

	if err != nil {
		log.Fatalf("client: loadkeys: %s", err)
	}

	// Create a CA certificate pool and add server CA to it
	caCert, err := ioutil.ReadFile("cert/ca/ca-cert.pem")
	if err != nil {
		log.Fatalf("client: read ca: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration with the client certificate and server's CA
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	// Set up the WebSocket dialer with the TLS configuration
	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	fmt.Print(dialer)
	conn, _, err := dialer.Dial("wss://localhost:8080", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := conn.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
