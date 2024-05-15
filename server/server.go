package main

import (
	"example/zerochat/chatProto"
	"fmt"
	_ "net/http/pprof"
)

func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case chatProto.CMD_SEND_MSG_SINGLE:
		fmt.Printf("FROM %s TO %s CONTENT: %s\n", msg.Sender, msg.Receipient, msg.Content)
	case chatProto.CMD_GET_USERS:
		fmt.Printf("GET USERS TRIGGERED BY %s\n", msg.Sender)
		chatProto.GetUsers(msg)
	}
}

func main() {
	chatProto.StartChatServer("127.0.0.1:8080", msgHandler)
}
