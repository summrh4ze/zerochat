package main

import (
	"example/zerochat/chatProto/domain"
	"example/zerochat/client/config"
	"example/zerochat/client/ui"
	"fmt"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/notify"
)

var (
	window  *app.Window
	focused bool
)

func repaint() {
	if window != nil {
		window.Invalidate()
	}
}

func run(window *app.Window, cfg config.Config, client *domain.Client) error {
	theme := material.NewTheme()
	usrChangedChan := make(chan string)
	var usersPanel *ui.UsersPanel
	var chatPanel *ui.ChatPanel
	var profilePanel *ui.ProfilePanel

	img, err := ui.CreateDefaultImage()
	if err != nil {
		log.Printf("failed to generate default avatar %s\n", err)
	}

	profilePanel = &ui.ProfilePanel{
		Avatar: img,
		OnConfirm: func(nickName string) {
			client = domain.InitClientConnection(nickName, img, cfg, func(err error) {
				if err != nil {
					usersPanel.ConnError = true
				}
				repaint()
			})
			usersPanel = ui.CreateUsersPanel(client, usrChangedChan)
			chatPanel = ui.CreateChatPanel(client, usrChangedChan)
		},
		OnImageLoad: func(image []byte) {
			img = image
			repaint()
		},
	}

	n, err := notify.NewNotifier()
	if err != nil {
		return fmt.Errorf("failed init notification manager: %s", err)
	}
	_, ongoingSupported := n.(notify.OngoingNotifier)
	notifier := n

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			if client == nil {
				profilePanel.Layout(gtx, theme)
			} else {
				chatScreen(gtx, theme, usersPanel, chatPanel)
			}

			if !focused && client != nil {
				for _, notif := range client.Notifications {
					if ongoingSupported {
						go notifier.(notify.OngoingNotifier).CreateOngoingNotification(notif.User.Name, notif.Message)
					} else {
						go notifier.CreateNotification(notif.User.Name, notif.Message)
					}
				}
				client.Notifications = client.Notifications[0:0]
			}

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)
		case app.ConfigEvent:
			focused = e.Config.Focused
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
	f, err := os.OpenFile("zerochat_client.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	cfg := config.ReadClientConfig()

	go func() {
		window = new(app.Window)
		window.Option(
			app.Title("Zerochat"),
			app.Size(unit.Dp(800), unit.Dp(500)),
			app.MinSize(unit.Dp(600), unit.Dp(400)),
		)
		err := run(window, cfg, nil)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
