package types

import (
	"sync"
	"time"
)

type Message struct {
	Sender     *User
	Receiver   *User
	Content    string
	Attachment []byte
	Timestamp  time.Time
}

type UserDetails struct {
	Id   string
	Name string
}

type User struct {
	UserDetails UserDetails
	Messages    []Message
}

type UserEvent interface {
	GetUserId() string
}

type UserDisconnectedEvent struct {
	Id string
}

type UserConnectedEvent struct {
	UserDetails UserDetails
}

func (ude UserDisconnectedEvent) GetUserId() string {
	return ude.Id
}

func (uce UserConnectedEvent) GetUserId() string {
	return uce.UserDetails.Id
}

type Registry struct {
	users     map[string]User
	EventChan chan UserEvent
	mutex     sync.Mutex
}

func InitRegistry(userDetails []UserDetails) *Registry {
	reg := Registry{
		users:     make(map[string]User),
		EventChan: make(chan UserEvent),
	}
	for _, u := range userDetails {
		reg.users[u.Id] = User{UserDetails: u, Messages: make([]Message, 0)}
	}

	go func() {
		for e := range reg.EventChan {
			switch event := e.(type) {
			case UserConnectedEvent:
				reg.users[e.GetUserId()] = User{
					UserDetails: event.UserDetails,
					Messages:    make([]Message, 0),
				}
			case UserDisconnectedEvent:
				delete(reg.users, e.GetUserId())
			}
		}
	}()

	return &reg
}

func (r *Registry) GetUserDetails() []UserDetails {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	res := make([]UserDetails, 0, len(r.users))
	for _, u := range r.users {
		res = append(res, u.UserDetails)
	}
	return res
}
