package main

import (
	"example/zerochat/chatProto"
	"example/zerochat/client/types"
	"example/zerochat/client/ui"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"regexp"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
)

var (
	id       = uuid.New().String()
	name     string
	registry *types.Registry
	window   *app.Window
	blue     = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}
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
		users := make([]types.UserDetails, 0, len(respUsers))
		for _, client := range respUsers {
			respUserDetails := strings.Split(client, ",")
			if len(respUserDetails) != 2 {
				fmt.Println("ERROR: Got client in incorrect format. Ignoring...")
				continue
			}
			users = append(users, types.UserDetails{Name: respUserDetails[0], Id: respUserDetails[1]})
		}
		registry = types.InitRegistry(users)
	case "conn_closed":
		chatProto.ClientQuit(id)
	case chatProto.CMD_SEND_MSG_SINGLE:
		fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
	case chatProto.CMD_USER_CONNECTED:
		fmt.Printf("User %s connected\n", msg.Content)
		if registry != nil {
			ud := strings.Split(msg.Content, ",")
			if len(ud) != 2 {
				fmt.Println("Error: Got client in incorrect format. Ignoring...")
				return
			}
			registry.EventChan <- types.UserConnectedEvent{UserDetails: types.UserDetails{Id: ud[1], Name: ud[0]}}
		}
	case chatProto.CMD_USER_DISCONNECTED:
		fmt.Printf("User with id %s disconnected\n", msg.Content)
		if registry != nil {
			registry.EventChan <- types.UserDisconnectedEvent{Id: msg.Content}
		}
	}
	repaint()
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	usersPanel := ui.CreateUsersPanel(registry, types.UserDetails{Id: id, Name: name})

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			chatLayout(gtx, theme, usersPanel)

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)
		}
	}
}

func ColorBox(gtx layout.Context, size image.Point, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func chatLayout(gtx layout.Context, theme *material.Theme, usersPanel *ui.UsersPanel) layout.Dimensions {
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
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ColorBox(gtx, gtx.Constraints.Min, blue)
		}),
	)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: client <name>")
		os.Exit(1)
	}
	name = os.Args[1]
	re := regexp.MustCompile(`^[0-9A-Za-z_]*$`)
	if !re.MatchString(name) {
		fmt.Println("Name can only contain 0-9 A-Z a-z and _")
		os.Exit(1)
	}

	err := chatProto.ConnectToChatServer("127.0.0.1", 8080, name, id, msgHandler)
	if err != nil {
		fmt.Printf("Error connecting to chat server: %s\n", err)
		os.Exit(1)
	}

	// First command is to get all users
	// After this command the server will send messages when clients connect or disconnect
	msg := chatProto.Message{
		Type:       "CMD_GET_USERS",
		Content:    "",
		Sender:     fmt.Sprintf("%s,%s", name, id),
		Receipient: "",
	}
	chatProto.ClientSendMsg(msg, id)

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
