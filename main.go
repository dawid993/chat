package main

import (
	"github.com/dawid993/goChat/chatServer"
	"github.com/dawid993/goChat/model"
)

func main() {
	s := chatServer.NewServer("localhost", 8080)
	s.Run(model.NewRoom("Chat"))
}
