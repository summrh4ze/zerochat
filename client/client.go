package main

import (
	"bufio"
	"example/zerochat/chatProto"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var id = uuid.New().String()

func msgHandler(msg chatProto.Message) {
	switch msg.Type {
	case chatProto.CMD_GET_USERS_RESPONSE:
		fmt.Printf("%s", msg.Content)
	case "conn_closed":
		chatProto.ClientQuit(id)
	case "msg":
		fmt.Printf("\n%s: %s\n", msg.Sender, msg.Content)
	}

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

	// read user input and write events to the channel
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
	fmt.Println("OUT!")
}
