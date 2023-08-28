package model

import (
	"context"

	"github.com/dawid993/goChat/db"
)

type Room struct {
	Name       string
	Clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Db         *db.DatabaseHandler
}

func (r *Room) Run(ctx context.Context) {
	for {
		select {
		case client := <-r.Register:
			r.Clients[client] = true
		case client := <-r.Unregister:
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				close(client.Message)
			}
		case message := <-r.Broadcast:
			for client := range r.Clients {
				select {
				case client.Message <- message:
				default:
					close(client.Message)
					delete(r.Clients, client)
				}
			}

			r.Db.InsertMessage(ctx, message.From, message.Content)
		}
	}
}

func NewRoom(name string) *Room {
	return &Room{
		Name:       name,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}
