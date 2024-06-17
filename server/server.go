package main

import (
	"encoding/json"
	"example/zerochat/chatProto"
	"example/zerochat/chatProto/domain"
	"example/zerochat/client/config"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"slices"
	"sync"

	"github.com/gorilla/websocket"
)

type hub struct {
	mutex   sync.Mutex
	clients map[string]*client
}

type client struct {
	user      *domain.User
	writeChan chan *domain.Message
}

func InitHub() *hub {
	return &hub{
		clients: make(map[string]*client),
	}
}

func (hub *hub) addClient(client *client) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	hub.clients[client.user.Id] = client
	for id, cli := range hub.clients {
		if id != client.user.Id {
			cli.writeChan <- &domain.Message{
				Type:   chatProto.CMD_USER_CONNECTED,
				Sender: *client.user,
			}
		}
	}
}

func (hub *hub) removeClient(client *client) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	delete(hub.clients, client.user.Id)
	for _, cli := range hub.clients {
		cli.writeChan <- &domain.Message{
			Type:   chatProto.CMD_USER_DISCONNECTED,
			Sender: *client.user,
		}
	}
}

func (hub *hub) getClient(user *domain.User) *client {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if c, ok := hub.clients[user.Id]; !ok {
		log.Printf("sender with id %s and name %s is not registered\n", user.Id, user.Name)
		return nil
	} else {
		return c
	}
}

func (hub *hub) getActiveUsers(message *domain.Message) (*domain.Message, error) {
	//log.Printf("GET USERS TRIGGERED BY %s\n", message.Sender.Name)
	if sender := hub.getClient(&message.Sender); sender == nil {
		return nil, fmt.Errorf("failed to return users. Sender not found")
	} else {
		hub.mutex.Lock()
		defer hub.mutex.Unlock()

		users := make([]*domain.User, 0, len(hub.clients))
		for _, cli := range hub.clients {
			if cli.user.Id != sender.user.Id {
				users = append(users, cli.user)
			}
		}
		var resp domain.Message
		resp.Sender = *sender.user
		resp.Type = chatProto.CMD_GET_USERS_RESPONSE
		// sort users alphabetically
		slices.SortFunc(users, func(a, b *domain.User) int {
			if a.Name < b.Name {
				return -1
			} else if a.Name > b.Name {
				return 1
			}
			return 0
		})
		content, err := json.Marshal(&users)
		if err != nil {
			return nil, fmt.Errorf("failed to marshall msg into json %s", err)
		}
		resp.Content = content
		return &resp, nil
	}
}

func (hub *hub) forwardMessage(message *domain.Message) {
	if sender := hub.getClient(&message.Sender); sender == nil {
		return
	} else {
		if receiver := hub.getClient(&message.Reciever); receiver == nil {
			log.Printf("failed to send message. Receiver does not exist")
			return
		} else {
			receiver.writeChan <- message
		}
	}
}

func (hub *hub) startChatServer(addr string) {
	log.Printf("chat server listening on %s\n", addr)

	var upgrader = websocket.Upgrader{}

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		// upgrade the connection from http to websocket
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %s\n", err)
			return
		}
		defer c.Close()

		// get the user from the client and save it
		var user domain.User
		err = c.ReadJSON(&user)
		if err != nil {
			log.Printf("failed reading user data %s\n", err)
			return
		}

		client := &client{
			user:      &user,
			writeChan: make(chan *domain.Message),
		}
		hub.addClient(client)

		// this gorutine checks if other clients want to send message to this connection
		// and if so it will send them
		go func() {
			for msg := range client.writeChan {
				err := c.WriteJSON(msg)
				if err != nil {
					log.Printf("failed writing json to websocket: %s\n", err)
					break
				}
			}
		}()

		// in this loop we read messages from clients and process them
		var message domain.Message
		for {
			err := c.ReadJSON(&message)
			if err != nil {
				log.Printf("failed websocket read: %s\n", err)
				break
			}
			switch message.Type {
			case chatProto.CMD_GET_USERS:
				resp, err := hub.getActiveUsers(&message)
				if err != nil {
					log.Printf("failed to get active users %s\n", err)
					continue
				}
				c.WriteJSON(resp)
			case chatProto.CMD_SEND_MSG_SINGLE:
				//log.Printf("SEND MESSAGE TRIGGERED BY %s TO %s\n", message.Sender.Name, message.Reciever.Name)
				hub.forwardMessage(&message)
			}
		}
		hub.removeClient(client)
		close(client.writeChan)
	})

	http.ListenAndServe(addr, nil)
}

func main() {
	f, err := os.OpenFile("zerochat_server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	cfg := config.ReadServerConfig()
	hub := InitHub()
	hub.startChatServer(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port))
}
