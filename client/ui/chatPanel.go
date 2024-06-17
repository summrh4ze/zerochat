package ui

import (
	"example/zerochat/chatProto"
	"example/zerochat/chatProto/domain"
	"fmt"
	"image/color"
	"log"
	"slices"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

var (
	grey = color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF}
	red  = color.NRGBA{R: 0xC0, G: 0x20, B: 0x20, A: 0xFF}
)

type ChatPanel struct {
	client            *domain.Client
	selectedUser      *domain.User
	input             component.TextField
	changeUserChannel <-chan string
	list              widget.List
}

func CreateChatPanel(
	client *domain.Client,
	changeUserChannel <-chan string,
) *ChatPanel {
	chatPanel := &ChatPanel{
		client:            client,
		changeUserChannel: changeUserChannel,
		selectedUser:      client.User,
	}

	go func() {
		for id := range chatPanel.changeUserChannel {
			log.Printf("got change selected user event %s\n", id)
			res, ok := client.ActiveUsers[id]
			if ok {
				chatPanel.selectedUser = res
			} else {
				chatPanel.selectedUser = client.User
			}
		}
	}()

	return chatPanel
}

func (chat *ChatPanel) getMessages() []*domain.Message {
	var messages []*domain.Message
	if chat.selectedUser.Id == chat.client.User.Id {
		messages = chat.client.Draft
	} else {
		chatHistory, ok := chat.client.ChatHistory[chat.selectedUser.Id]
		if !ok {
			return []*domain.Message{}
		}
		messages = chatHistory.Messages
	}
	slices.SortFunc(messages, func(a, b *domain.Message) int {
		if a.Timestamp.Before(b.Timestamp) {
			return -1
		} else if a.Timestamp.After(b.Timestamp) {
			return 1
		}
		return 1
	})

	//update chathistory unread to false
	history := chat.client.ChatHistory[chat.selectedUser.Id]
	history.Unread = false
	chat.client.ChatHistory[chat.selectedUser.Id] = history

	return messages
}

func (chat *ChatPanel) processEvents(gtx layout.Context) {
	for {
		e, ok := chat.input.Editor.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			t := e.Text

			// clear the input
			chat.input.SetText("")
			if t == "" {
				return
			}
			if chat.selectedUser.Id != chat.client.User.Id {
				msg := &domain.Message{
					Type:     chatProto.CMD_SEND_MSG_SINGLE,
					Sender:   *chat.client.User,
					Reciever: *chat.selectedUser,
					Content:  []byte(t),
				}
				chat.client.WriteChan <- msg
				history := chat.client.ChatHistory[chat.selectedUser.Id]
				history.Messages = append(history.Messages, msg)
				chat.client.ChatHistory[chat.selectedUser.Id] = history
			} else {
				msg := &domain.Message{
					Type:     chatProto.CMD_SEND_MSG_SINGLE,
					Sender:   *chat.client.User,
					Reciever: *chat.client.User,
					Content:  []byte(t),
				}
				chat.client.Draft = append(chat.client.Draft, msg)
			}
		}
	}
}

func (chat *ChatPanel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	messages := chat.getMessages()
	chat.processEvents(gtx)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := material.Label(theme, unit.Sp(18), chat.selectedUser.Name)
				lb.Font.Weight = font.Bold
				return lb.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			chat.list.Axis = layout.Vertical
			chat.list.ScrollToEnd = true
			return chat.list.Layout(gtx, len(messages), func(gtx layout.Context, index int) layout.Dimensions {
				max := len(chat.selectedUser.Name)
				if len(chat.selectedUser.Name) < len(chat.client.User.Name) {
					max = len(chat.client.User.Name)
				}
				max += 3

				display := fmt.Sprintf(
					"%-*s%s",
					max,
					messages[index].Sender.Name,
					//messages[index].Timestamp.Format(time.Kitchen),
					messages[index].Content,
				)
				lb := material.Label(theme, unit.Sp(16), display)
				if messages[index].Sender.Id == chat.client.User.Id {
					lb.Color = grey
				} else {
					lb.Color = red
				}
				lb.Font.Typeface = "Consolas"
				return lb.Layout(gtx)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			chat.input.Submit = true
			return chat.input.Layout(gtx, theme, "Enter Message")
		}),
	)
}
