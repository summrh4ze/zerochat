package main

import (
	"example/zerochat/chatProto"
	"example/zerochat/client/config"
	"fmt"
	_ "net/http/pprof"
)

func msgHandler(msg chatProto.Message) {
	fmt.Printf("server msgHandler %#v\n", msg)
	switch msg.Type {
	case chatProto.CMD_GET_USERS:
		fmt.Printf("GET USERS TRIGGERED BY %s\n", msg.Sender)
		chatProto.GetUsers(msg)
	case chatProto.CMD_SEND_MSG_SINGLE:
		fmt.Printf("SEND MESSAGE SINGLE TRIGGERED BY %s TO %s\n", msg.Sender, msg.Receipient)
		chatProto.SendMessage(msg)
	}
}

func main() {
	cfg := config.ReadServerConfig()
	chatProto.StartChatServer(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), msgHandler)
}
