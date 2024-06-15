package main

import (
	"example/zerochat/chatProto"
	"example/zerochat/client/config"
	"example/zerochat/client/ui"
	"example/zerochat/client/users"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
)

var (
	id       = uuid.New().String()
	name     string
	icon     image.Image
	registry *users.Registry
	window   *app.Window
)

func repaint() {
	if window != nil {
		window.Invalidate()
	}
}

// this runs on other thread
func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case chatProto.CMD_GET_USERS_RESPONSE:
		respUsers := strings.Split(msg.Content, "\n")
		onlineUsers := make([]users.UserDetails, 0, len(respUsers))
		for _, client := range respUsers {
			respUserDetails := strings.Split(client, ",")
			if len(respUserDetails) != 2 {
				fmt.Println("ERROR: Got client in incorrect format. Ignoring...")
				continue
			}
			onlineUsers = append(onlineUsers, users.UserDetails{Name: respUserDetails[0], Id: respUserDetails[1], Avatar: icon})
		}
		registry = users.InitRegistry(onlineUsers, users.UserDetails{Name: name, Id: id, Avatar: icon})
	case "conn_closed":
		chatProto.ClientQuit(id)
	case chatProto.CMD_SEND_MSG_SINGLE:
		fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
		if registry != nil {
			senderDetails := strings.Split(msg.Sender, ",")
			if len(senderDetails) != 2 {
				fmt.Println("Error: Message Sender incorrect format. Ignoring...")
				return
			}
			sender, ok := registry.GetUserById(senderDetails[1])
			if !ok {
				fmt.Println("Message was sent by unknown user. Ignoring...")
				return
			}
			registry.EventChan <- users.AddMessageToUserEvent{
				Id: senderDetails[1],
				Msg: users.Message{
					Sender:    sender.UserDetails,
					Content:   msg.Content,
					Timestamp: time.Now(),
				}}
		}
	case chatProto.CMD_USER_CONNECTED:
		fmt.Printf("User %s connected\n", msg.Content)
		if registry != nil {
			ud := strings.Split(msg.Content, ",")
			if len(ud) != 2 {
				fmt.Println("Error: Got client in incorrect format. Ignoring...")
				return
			}
			registry.EventChan <- users.UserConnectedEvent{UserDetails: users.UserDetails{Id: ud[1], Name: ud[0], Avatar: icon}}
		}
	case chatProto.CMD_USER_DISCONNECTED:
		fmt.Printf("User with id %s disconnected\n", msg.Content)
		if registry != nil {
			registry.EventChan <- users.UserDisconnectedEvent{Id: msg.Content}
		}
	}
	repaint()
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	usrChangedChan := make(chan string)
	usersPanel := ui.CreateUsersPanel(registry, users.UserDetails{Id: id, Name: name, Avatar: icon}, usrChangedChan)
	chatPanel := ui.CreateChatPanel(
		registry,
		users.UserDetails{Id: id, Name: name, Avatar: icon},
		usrChangedChan,
		users.UserDetails{Id: id, Name: name, Avatar: icon},
	)

	if img, err := ui.CreateDefaultImage(); err == nil {
		icon = img
	}

	profilePanel := ui.ProfilePanel{
		Avatar: icon,
		OnConfirm: func(nick string) {
			name = nick

			// First connect to the client
			cfg := config.ReadClientConfig()
			err := chatProto.ConnectToChatServer(cfg.Host, cfg.Port, name, id, msgHandler)
			if err != nil {
				fmt.Printf("Error connecting to chat server: %s\n", err)
				os.Exit(1)
			}

			// Then execute first command to get all users
			// After this command the server will send messages when clients connect or disconnect
			msg := chatProto.Message{
				Type:       "CMD_GET_USERS",
				Content:    "",
				Sender:     fmt.Sprintf("%s,%s", name, id),
				Receipient: "",
			}
			chatProto.ClientSendMsg(msg, id)
		},
		OnImageLoad: func(image image.Image) {
			fmt.Println("Got image. Repainting...")
			icon = image
			repaint()
		},
	}

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			if name == "" {
				profilePanel.Layout(gtx, theme)
			} else {
				chatScreen(gtx, theme, usersPanel, chatPanel)
			}

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)
		}
	}
}

func chatScreen(
	gtx layout.Context,
	theme *material.Theme,
	usersPanel *ui.UsersPanel,
	chatPanel *ui.ChatPanel,
) layout.Dimensions {
	return layout.Flex{}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			border := widget.Border{Color: color.NRGBA{A: 0xff}, Width: unit.Dp(2)}
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return usersPanel.Layout(gtx, theme)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return chatPanel.Layout(gtx, theme)
			})
		}),
	)
}

func main() {
	go func() {
		window = new(app.Window)
		window.Option(
			app.Title("Zerochat"),
			app.Size(unit.Dp(800), unit.Dp(500)),
			app.MinSize(unit.Dp(600), unit.Dp(400)),
		)
		err := run(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
