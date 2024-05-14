package main

import (
	"bufio"
	"example/zerochat/chatProto"
	"fmt"
	"os"
	"strings"
	"time"
)

func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case "conn_closed":
		chatProto.Quit()
	case "msg":
		fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
	}

}

func main() {
	err := chatProto.ConnectToChatServer("127.0.0.1", 8080, msgHandler)
	if err != nil {
		fmt.Printf("Error connecting to chat server: %s\n", err)
		os.Exit(1)
	}

	// read user input and write events to the channel
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading user input: %s\n", err)
			os.Exit(1)
		}
		// Remove the delimiter from the string
		msg = strings.Trim(msg, "\r\n")
		if msg == "quit" {
			fmt.Println("QUITTING.........")
			chatProto.Quit()
			break
		} else {
			chatProto.SendMsg(chatProto.Message{Type: "cmd_send", Content: msg, Sender: "???", ChatRoom: "public"})
		}
	}

	time.Sleep(5 * time.Second)
	fmt.Println("OUT!")
}
