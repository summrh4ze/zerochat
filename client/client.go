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
)

var (
	window *app.Window
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
		fmt.Printf("failed to generate default avatar %s\n", err)
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
			fmt.Println("Got image. Repainting...")
			img = image
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

			if client == nil {
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
