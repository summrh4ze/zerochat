package types

import (
	"fmt"
	"sync"
	"time"
)

type Message struct {
	Sender    UserDetails
	Content   string
	Timestamp time.Time
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

type AddMessageToUserEvent struct {
	Id  string
	Msg Message
}

type AddMessageToSelfEvent struct {
	Id  string
	Msg Message
}

func (ude UserDisconnectedEvent) GetUserId() string {
	return ude.Id
}

func (uce UserConnectedEvent) GetUserId() string {
	return uce.UserDetails.Id
}

func (mse AddMessageToUserEvent) GetUserId() string {
	return mse.Id
}

func (mss AddMessageToSelfEvent) GetUserId() string {
	return mss.Id
}

type Registry struct {
	users     map[string]User
	self      User
	EventChan chan UserEvent
	mutex     sync.Mutex
}

func InitRegistry(userDetails []UserDetails, self UserDetails) *Registry {
	fmt.Printf("\n\n............INITIALIZING REGISTRY...............\n\n\n")
	reg := Registry{
		users:     make(map[string]User),
		self:      User{UserDetails: self, Messages: make([]Message, 0)},
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
			case AddMessageToUserEvent:
				user := reg.users[e.GetUserId()]
				user.Messages = append(user.Messages, event.Msg)
				reg.users[e.GetUserId()] = user
			case AddMessageToSelfEvent:
				reg.self.Messages = append(reg.self.Messages, event.Msg)
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

func (r *Registry) GetUserById(id string) (User, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	u, ok := r.users[id]
	return u, ok
}

func (r *Registry) GetSelf() User {
	return r.self
}
