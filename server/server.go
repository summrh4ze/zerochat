package main

import (
	"example/zerochat/chatProto"
	"fmt"
)

func main() {
	messageChannel, _ := chatProto.StartChatServer(":8080")

	// only display the messages received from the client
	fmt.Println("Running loop to display on UI the received messages")
	for {
		msg := <-messageChannel
		fmt.Printf("HOURRAY: You got: %s\n", msg)
	}
}
