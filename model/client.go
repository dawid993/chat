package model

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
)

type Member struct {
	Name    string
	Profile MemberProfile
}

type MemberProfile struct {
	Age       int
	Location  string
	Interests []string
}

type Message struct {
	From    string
	Content string
}

type Client struct {
	Conn    *websocket.Conn
	Member  Member
	Message chan Message
	Room    *Room
	Closed  chan bool
}

func (c *Client) Read(ctx context.Context) {
	clientCtx, clientCtxCancel := context.WithCancel(ctx)
	defer clientCtxCancel()

	for {
		select {
		case <-clientCtx.Done():

			c.Room.Unregister <- c
			c.Conn.Close()
			fmt.Println("Client read context done - in")

			c.Closed <- true

			fmt.Println("Client read context done - out")
			return
		default:
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				c.Room.Unregister <- c
				c.Conn.Close()
				c.Closed <- true
				return
			}
			c.Room.Broadcast <- Message{c.Member.Name, string(message)}
		}

	}
}

func (c *Client) Write(ctx context.Context) {
	clientCtx, clientCtxCancel := context.WithCancel(ctx)
	defer clientCtxCancel()

	for message := range c.Message {
		select {
		case <-clientCtx.Done():
			c.Room.Unregister <- c
			c.Conn.Close()
			fmt.Println("Client write context done - in")

			c.Closed <- true

			fmt.Println("Client write context done - out")

			return
		default:
			err := c.Conn.WriteJSON(message)
			if err != nil {
				c.Room.Unregister <- c
				c.Conn.Close()
				return
			}
		}

	}
}
