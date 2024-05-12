package main

import (
	"example/zerochat/websocket"
	"fmt"
)

func main() {
	conn, err := websocket.Connect("ws://127.0.0.1:8080/chat")
	if err != nil {
		fmt.Printf("Error connecting to websocket: %s\n", err)
	}

	defer conn.Close()
}
