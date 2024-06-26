package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Type      string
	Sender    User
	Reciever  User
	Content   []byte
	Timestamp time.Time
}

type User struct {
	Id     string
	Name   string
	Avatar []byte
}

type Notification struct {
	User    *User
	Message string
}

type ChatHistory struct {
	Messages []*Message
	Unread   bool
}

func CreateUser(nickName string, avatar []byte) *User {
	return &User{Id: uuid.New().String(), Name: nickName, Avatar: avatar}
}

func (u *User) String() string {
	return fmt.Sprintf("%s:%s", u.Id, u.Name)
}

func (n *Notification) String() string {
	return fmt.Sprintf("%s: %s", n.User.Name, n.Message)
}
