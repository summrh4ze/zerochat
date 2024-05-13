package main

import (
	ws "example/zerochat/websocket"
	"fmt"
	"os"
)

func main() {
	conn, err := ws.Connect("ws://127.0.0.1:8080/chat")
	if err != nil {
		fmt.Printf("Error connecting to websocket: %s\n", err)
		os.Exit(1)
	}

	defer conn.Close()
}
