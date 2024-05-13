package main

import (
	"example/zerochat/chatProto"
	"fmt"
	"os"
)

func main() {
	messageChannel, eventChannel, err := chatProto.ConnectToChatServer("127.0.0.1", 8080)
	if err != nil {
		fmt.Printf("Error connecting to chat server: %s\n", err)
		os.Exit(1)
	}

	shouldQuit := false

	// read from message channel and display the message
	go func() {
		for !shouldQuit {
			msg := <-messageChannel
			fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
		}
		fmt.Println("Exiting from read messages loop")
	}()

	// read user input and write events to the channel
	var msg string
	for !shouldQuit {
		_, err := fmt.Scanln(&msg)
		if err != nil {
			fmt.Printf("Error reading user input: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("You have entered %s\n", msg)
		if msg == "quit" {
			shouldQuit = true
		}
		eventChannel <- chatProto.Event{Type: "msg", Msg: msg}
	}

	fmt.Println("OUT!")
}
