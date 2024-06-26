package domain

import (
	"encoding/json"
	"example/zerochat/chatProto"
	"example/zerochat/client/config"
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

type Client struct {
	User          *User
	Draft         []*Message
	WriteChan     chan *Message
	ActiveUsers   map[string]*User
	ChatHistory   map[string]ChatHistory
	Notifications []Notification
}

func (client *Client) connectToChatServer(hostPort string, callback func(error)) {
	u := url.URL{Scheme: "ws", Host: hostPort, Path: "/chat"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("failed to dial websocket server %s", err)
		callback(err)
		return
	}
	defer c.Close()

	// send the current user to server
	c.WriteJSON(client.User)

	go func() {
		for msg := range client.WriteChan {
			err := c.WriteJSON(msg)
			if err != nil {
				log.Printf("failed writing json to websocket: %s\n", err)
				break
			}
		}
		log.Println("exit gorutine that reads from write channel")
	}()

	for {
		var message Message
		err := c.ReadJSON(&message)
		if err != nil {
			log.Printf("failed to read %s\n", err)
			break
		}

		switch message.Type {
		case chatProto.CMD_GET_USERS_RESPONSE:
			log.Printf("request current active users on server\n")
			users := make([]*User, 0)
			err := json.Unmarshal(message.Content, &users)
			if err != nil {
				log.Printf("failed to unmarshall json with users %s\n", err)
				continue
			}
			clear(client.ActiveUsers)
			for _, u := range users {
				client.ActiveUsers[u.Id] = u
			}
			log.Printf("active users: %v\n", client.ActiveUsers)
		case chatProto.CMD_SEND_MSG_SINGLE:
			log.Printf("got message from %s:%s\n", message.Sender.Id, message.Sender.Name)
			history := client.ChatHistory[message.Sender.Id]
			history.Messages = append(
				history.Messages,
				&message,
			)
			history.Unread = true
			client.ChatHistory[message.Sender.Id] = history
			notif := Notification{User: &message.Sender, Message: string(message.Content)}
			client.Notifications = append(client.Notifications, notif)
		case chatProto.CMD_USER_CONNECTED:
			log.Printf("User %s:%s connected\n", message.Sender.Id, message.Sender.Name)
			client.ActiveUsers[message.Sender.Id] = &message.Sender
			client.ChatHistory[message.Sender.Id] = ChatHistory{
				Messages: make([]*Message, 0),
				Unread:   false,
			}
			log.Printf("active users: %v\n", client.ActiveUsers)
		case chatProto.CMD_USER_DISCONNECTED:
			log.Printf("User %s:%s disconnected\n", message.Sender.Id, message.Sender.Name)
			delete(client.ActiveUsers, message.Sender.Id)
			delete(client.ChatHistory, message.Sender.Id)
			log.Printf("active users: %v\n", client.ActiveUsers)
		}
		callback(nil)
	}
	close(client.WriteChan)
	log.Println("exit gorutine connectToChatServer")
}

func InitClientConnection(
	nickName string,
	avatar []byte,
	cfg config.Config,
	callback func(error),
) *Client {
	// First connect to the client
	hostPort := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	user := CreateUser(nickName, avatar)
	client := &Client{
		User:        user,
		Draft:       make([]*Message, 0),
		WriteChan:   make(chan *Message, 1),
		ActiveUsers: make(map[string]*User),
		ChatHistory: make(map[string]ChatHistory),
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
