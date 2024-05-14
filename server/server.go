package main

import (
	"example/zerochat/chatProto"
	"fmt"
	_ "net/http/pprof"
)

func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case "cmd_send":
		fmt.Printf("%s: %s\n", msg.Sender, msg.Content)
	}

}

func main() {
	chatProto.StartChatServer("127.0.0.1:8080", msgHandler)
}
