package ui

import (
	"example/zerochat/chatProto"
	"example/zerochat/client/types"
	"fmt"
	"image/color"
	"sort"
	"time"

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
	clientDetails     types.UserDetails
	registry          *types.Registry
	input             component.TextField
	selectedUser      types.UserDetails
	changeUserChannel <-chan string
	list              widget.List
}

func CreateChatPanel(
	registry *types.Registry,
	selectedUser types.UserDetails,
	changeUserChannel <-chan string,
	clientDetails types.UserDetails,
) *ChatPanel {
	chatPanel := &ChatPanel{
		clientDetails:     clientDetails,
		registry:          registry,
		selectedUser:      selectedUser,
		changeUserChannel: changeUserChannel,
	}

	go func() {
		for id := range chatPanel.changeUserChannel {
			if registry != nil {
				res, ok := registry.GetUserById(id)
				if ok {
					chatPanel.selectedUser = res.UserDetails
				}
			}
		}
	}()

	return chatPanel
}

func (chat *ChatPanel) getMessages() []types.Message {
	if chat.registry != nil {
		var messages []types.Message
		if chat.selectedUser.Id == chat.clientDetails.Id {
			user := chat.registry.GetSelf()
			messages = user.Messages
		} else {
			user, ok := chat.registry.GetUserById(chat.selectedUser.Id)
			if !ok {
				return []types.Message{}
			}
			messages = user.Messages
		}
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Timestamp.Before(messages[i].Timestamp)
		})
		return messages

	}
	return []types.Message{}
}

func (chat *ChatPanel) processEvents(gtx layout.Context) {
	for {
		e, ok := chat.input.Editor.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			t := e.Text
			fmt.Printf("GOT SUBMIT EVENT WITH TEXT: %s\n", t)

			// clear the input
			chat.input.SetText("")
			if t == "" {
				return
			}
			if chat.selectedUser.Id != chat.clientDetails.Id {
				msg := chatProto.Message{
					Type:       chatProto.CMD_SEND_MSG_SINGLE,
					Sender:     fmt.Sprintf("%s,%s", chat.clientDetails.Name, chat.clientDetails.Id),
					Receipient: fmt.Sprintf("%s,%s", chat.selectedUser.Name, chat.selectedUser.Id),
					Content:    t,
				}
				chatProto.ClientSendMsg(msg, chat.clientDetails.Id)
				chat.registry.EventChan <- types.AddMessageToUserEvent{
					Id: chat.selectedUser.Id,
					Msg: types.Message{
						Sender:    chat.clientDetails,
						Content:   t,
						Timestamp: time.Now(),
					},
				}
			} else {
				chat.registry.EventChan <- types.AddMessageToSelfEvent{
					Id: chat.clientDetails.Id,
					Msg: types.Message{
						Sender:    chat.clientDetails,
						Content:   t,
						Timestamp: time.Now(),
					},
				}
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
			return chat.list.Layout(gtx, len(messages), func(gtx layout.Context, index int) layout.Dimensions {
				max := len(chat.selectedUser.Name)
				if len(chat.selectedUser.Name) < len(chat.clientDetails.Name) {
					max = len(chat.clientDetails.Name)
				}
				max++

				display := fmt.Sprintf(
					"%-*s:%s",
					max,
					messages[index].Sender.Name,
					//messages[index].Timestamp.Format(time.Kitchen),
					messages[index].Content,
				)
				lb := material.Label(theme, unit.Sp(14), display)
				if messages[index].Sender.Id == chat.clientDetails.Id {
					lb.Color = grey
				} else {
					lb.Color = red
				}
				lb.Font.Typeface = "Consolas"
				return lb.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			chat.input.Submit = true
			return chat.input.Layout(gtx, theme, "Enter Message")
		}),
	)
}