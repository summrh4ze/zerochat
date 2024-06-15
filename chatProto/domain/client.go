package domain

import (
	"encoding/json"
	"example/zerochat/chatProto"
	"example/zerochat/client/config"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

type Client struct {
	User        *User
	Draft       []*Message
	WriteChan   chan *Message
	ActiveUsers map[string]*User
	ChatHistory map[string][]*Message
}

func (client *Client) connectToChatServer(hostPort string, callback func()) {
	u := url.URL{Scheme: "ws", Host: hostPort, Path: "/chat"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("failed to dial websocket server %s", err)
		return
	}
	defer c.Close()

	// send the current user to server
	c.WriteJSON(client.User)

	go func() {
		for msg := range client.WriteChan {
			err := c.WriteJSON(msg)
			if err != nil {
				fmt.Printf("failed writing json to websocket: %s\n", err)
				break
			}
		}
		fmt.Println("exiting write to connection loop")
	}()

	var message Message
	for {
		err := c.ReadJSON(&message)
		if err != nil {
			fmt.Printf("failed to read %s\n", err)
			break
		}
		//fmt.Printf("recv: %s", message)
		switch message.Type {
		case chatProto.CMD_GET_USERS_RESPONSE:
			users := make([]*User, 0)
			err := json.Unmarshal(message.Content, &users)
			if err != nil {
				fmt.Printf("failed to unmarshall json with users %s\n", err)
				continue
			}
			clear(client.ActiveUsers)
			for _, u := range users {
				client.ActiveUsers[u.Id] = u
			}
			fmt.Printf("clients activeUsers %v\n", client.ActiveUsers)
		case chatProto.CMD_SEND_MSG_SINGLE:
			fmt.Printf("got message from %s:%s\n", message.Sender.Id, message.Sender.Name)
			client.ChatHistory[message.Sender.Id] = append(
				client.ChatHistory[message.Sender.Id],
				&message,
			)
		case chatProto.CMD_USER_CONNECTED:
			fmt.Printf("User %s:%s connected\n", message.Sender.Id, message.Sender.Name)
			client.ActiveUsers[message.Sender.Id] = &message.Sender
			client.ChatHistory[message.Sender.Id] = make([]*Message, 0)
		case chatProto.CMD_USER_DISCONNECTED:
			fmt.Printf("User %s:%s disconnected\n", message.Sender.Id, message.Sender.Name)
			delete(client.ActiveUsers, message.Sender.Id)
			delete(client.ChatHistory, message.Sender.Id)
		}
		callback()
	}
	close(client.WriteChan)
}

func InitClientConnection(nickName string, avatar []byte, callback func()) *Client {
	// First connect to the client
	cfg := config.ReadClientConfig()
	hostPort := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	user := CreateUser(nickName, avatar)
	client := &Client{
		User:        user,
		Draft:       make([]*Message, 0),
		WriteChan:   make(chan *Message),
		ActiveUsers: make(map[string]*User),
		ChatHistory: make(map[string][]*Message),
	}
	go client.connectToChatServer(hostPort, callback)

	// Then execute first command to get all users
	// After this command the server will send messages when clients connect or disconnect
	msg := Message{
		Type:   chatProto.CMD_GET_USERS,
		Sender: *client.User,
	}
	client.WriteChan <- &msg
	return client
}
