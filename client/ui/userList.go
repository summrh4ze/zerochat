package ui

import (
	"example/zerochat/chatProto"
	"example/zerochat/chatProto/domain"
	"fmt"
	"image/color"
	"slices"
	"strings"

	"gioui.org/layout"
	"gioui.org/widget/material"
	"golang.org/x/exp/maps"
)

var blue = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}

type UserList struct {
	client            *domain.Client
	list              layout.List
	userCards         []*UserCard
	changeUserChannel chan<- string
}

func (list *UserList) processClickEvents(gtx layout.Context) {
	for i, card := range list.userCards {
		if card.btn.Clicked(gtx) {
			fmt.Printf("You clicked on item %d\n", i)
			list.changeUserChannel <- card.user.Id
		}
	}
}

func filterMessages(messages []*domain.Message) []*domain.Message {
	res := []*domain.Message{}
	for _, m := range messages {
		if m.Type == chatProto.CMD_SEND_MSG_SINGLE {
			res = append(res, m)
		}
	}
	return res
}

func (list *UserList) getLastMessage(user *domain.User) string {
	message := "Say Hi!"
	chatHistory := list.client.ChatHistory[user.Id]
	if len(chatHistory) > 0 {
		chatHistory = filterMessages(chatHistory)
		slices.SortFunc(chatHistory, func(a, b *domain.Message) int {
			if a.Timestamp.Before(b.Timestamp) {
				return -1
			} else if a.Timestamp.After(b.Timestamp) {
				return 1
			}
			return 0
		})
		if len(chatHistory) > 0 {
			return string(chatHistory[len(chatHistory)-1].Content)
		}
	}
	return message
}

func (list *UserList) updateUserCards() {
	fmt.Printf("!!!active users: %v\n", list.client.ActiveUsers)
	users := maps.Values(list.client.ActiveUsers)
	slices.SortFunc(users, func(a, b *domain.User) int {
		if strings.ToLower(a.Name) < strings.ToLower(b.Name) {
			return -1
		} else if strings.ToLower(a.Name) > strings.ToLower(b.Name) {
			return 1
		}
		return 0
	})

	usersLen := len(users)
	currentCardsLen := len(list.userCards)

	// increase the capacity in case the number of users grows
	if usersLen > currentCardsLen {
		buf := make([]*UserCard, usersLen-currentCardsLen)
		list.userCards = append(list.userCards, buf...)
	}

	// and rewrite each card to the corresponding user to preserve the order
	for i, user := range users {
		if i < len(list.userCards) {
			message := list.getLastMessage(user)
			if list.userCards[i] == nil {
				list.userCards[i] = &UserCard{
					user:    user,
					message: message,
				}
			} else {
				list.userCards[i].user = user
				list.userCards[i].message = message
			}
		}
	}

	list.userCards = list.userCards[:len(users)]
	fmt.Printf("cards: %v\n", list.userCards)
}

func (list *UserList) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	list.updateUserCards()
	list.processClickEvents(gtx)
	return list.list.Layout(gtx, len(list.userCards), func(gtx layout.Context, index int) layout.Dimensions {
		return list.userCards[index].Layout(gtx, theme)
	})
}
