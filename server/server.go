package main

import (
	"example/zerochat/chatProto"
	"fmt"
)

func main() {
	messageChannel, _ := chatProto.StartChatServer("127.0.0.1:8080")

	// only display the messages received from the client
	for {
		msg := <-messageChannel
		fmt.Printf("%s: %s\n", msg.Sender, msg.Content)
	}
}
