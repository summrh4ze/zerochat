package domain

import (
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

func CreateUser(nickName string, avatar []byte) *User {
	return &User{Id: uuid.New().String(), Name: nickName, Avatar: avatar}
}
