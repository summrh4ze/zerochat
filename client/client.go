package main

import (
	"example/zerochat/chatProto"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

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
	id            = uuid.New().String()
	onlineClients = make([]Client, 0, 10)
	window        *app.Window
	mutex         = sync.Mutex{}
	background    = color.NRGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xFF}
	red           = color.NRGBA{R: 0xC0, G: 0x40, B: 0x40, A: 0xFF}
	green         = color.NRGBA{R: 0x40, G: 0xC0, B: 0x40, A: 0xFF}
	blue          = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}
)

type Client struct {
	name string
	id   string
}

func repaint() {
	if window != nil {
		window.Invalidate()
	}
}

// this runs on other thread
func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case chatProto.CMD_GET_USERS_RESPONSE:
		clients := strings.Split(msg.Content, "\n")
		clientSlice := make([]Client, 0, len(clients))
		for _, client := range clients {
			clientDetails := strings.Split(client, ",")
			if len(clientDetails) != 2 {
				fmt.Println("ERROR: Got client in incorrect format. Ignoring...")
				return
			}
			clientSlice = append(clientSlice, Client{name: clientDetails[0], id: clientDetails[1]})
		}
		mutex.Lock()
		onlineClients = clientSlice
		mutex.Unlock()
		repaint()
	case "conn_closed":
		chatProto.ClientQuit(id)
	case chatProto.CMD_SEND_MSG_SINGLE:
		fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
	}
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			chatLayout(gtx, theme)

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

func UserCard(gtx layout.Context, theme *material.Theme, user Client) layout.Dimensions {
	border := widget.Border{Color: color.NRGBA{A: 0xff}, Width: unit.Dp(1)}
	return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = 300
			gtx.Constraints.Max.Y = 60
			dim := layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dim := layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						dim := gtx.Constraints.Max.Y
						gtx.Constraints.Max = image.Point{X: dim, Y: dim}
						circle := clip.Ellipse{Max: image.Pt(dim, dim)}.Op(gtx.Ops)
						paint.FillShape(gtx.Ops, blue, circle)
						return layout.Dimensions{Size: image.Pt(dim, dim)}
					})
					return dim
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					fmt.Printf("INSIDE tab label constraints = %#v\n", gtx.Constraints)
					return layout.Flex{Axis: layout.Vertical}.Layout(
						gtx,
						layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
							size := gtx.Constraints.Max
							//ColorBox(gtx, size, green)
							label := material.Label(theme, unit.Sp(16), user.name)
							label.MaxLines = 1
							label.Layout(gtx)
							return layout.Dimensions{Size: size}
						}),
						layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
							size := gtx.Constraints.Max
							//ColorBox(gtx, size, green)
							label := material.Label(theme, unit.Sp(12), "Say Hi!")
							label.MaxLines = 1
							label.Layout(gtx)
							return layout.Dimensions{Size: size}
						}),
					)
				}),
			)
			fmt.Printf("SIZE OF TAB FLEX = %#v\n", dim)
			return dim
		})
	})
}

func UsersPanel(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10), Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.H5(theme, "Users")
				return title.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			usersList := layout.List{Axis: layout.Vertical}
			mutex.Lock()
			dim := usersList.Layout(gtx, len(onlineClients), func(gtx layout.Context, index int) layout.Dimensions {
				return UserCard(gtx, theme, onlineClients[index])
			})
			mutex.Unlock()
			return dim
		}),
	)
}

func chatLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			border := widget.Border{Color: color.NRGBA{A: 0xff}, Width: unit.Dp(2)}
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return UsersPanel(gtx, theme)
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
	name := os.Args[1]
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

	go func() {
		for {
			msg := chatProto.Message{
				Type:       "CMD_GET_USERS",
				Content:    "",
				Sender:     fmt.Sprintf("%s,%s", name, id),
				Receipient: "",
			}
			chatProto.ClientSendMsg(msg, id)
			time.Sleep(100 * time.Second)
		}
	}()

	/* // read user input and write events to the channel
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading user input: %s\n", err)
			os.Exit(1)
		}
		// Remove the delimiter from the string
		msg = strings.Trim(msg, "\r\n")
		if msg == "quit" {
			fmt.Println("QUITTING.........")
			chatProto.ClientQuit(id)
			break
		} else {
			// input cmd::content::<recipient_name,receipient_id>
			var message chatProto.Message
			message.Sender = fmt.Sprintf("%s,%s", name, id)
			for i, part := range strings.Split(msg, "::") {
				switch i {
				case 0:
					message.Type = part
				case 1:
					message.Content = part
				case 2:
					message.Receipient = part
				}
			}
			chatProto.ClientSendMsg(message, id)
		}
	}

	time.Sleep(5 * time.Second)
	fmt.Println("OUT!") */

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
