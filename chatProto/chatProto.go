package chatProto

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/google/uuid"
)

var (
	sharedGUID = "d0bffdf3-9439-4776-9e5b-a52575d6ead7"
)

type Message struct {
	Content  string
	Sender   string
	ChatRoom string
}

type Event struct {
	Type string
	Msg  string
}

func runChatProtocol(conn net.Conn, messageChannel chan Message, eventChannel chan Event) {
	defer conn.Close()
	fmt.Println("Run Chat Protocol")

	shouldQuit := false

	// this is reading from the TCP connection and writing to the message channel
	go func() {
		fmt.Println("Running loop to read from TCP")
		buffer := make([]byte, 10000)
		for !shouldQuit {
			n, err := conn.Read(buffer)
			if err != nil {
				eventChannel <- Event{Type: "closed_conn"}
				fmt.Printf("Error %s\n", err)
				return
			} else {
				messageChannel <- Message{Content: string(buffer[0:n]), Sender: "???", ChatRoom: "public"}
			}
		}
	}()

	// this is reading from the eventChannel and writing to the TCP connection
	for !shouldQuit {
		fmt.Println("Running loop to read from eventChannel")
		ev := <-eventChannel
		if ev.Type == "quit" || ev.Type == "closed_conn" {
			shouldQuit = true
			return
		}
		fmt.Printf("Sending %s\n", ev)
		conn.Write([]byte(ev.Msg))
	}
	fmt.Println("Exiting from read eventChannel loop")
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

func handleChatMessages(conn net.Conn, messageChannel chan string, eventChannel chan Event) {
	defer conn.Close()
	fmt.Println("Handle chat messages")

	shouldQuit := false

	// here we are reading from TCP connection and writing to messageChannel
	go func() {
		fmt.Println("Running loop for reading from TCP connection")
		buffer := make([]byte, 10000)
		for !shouldQuit {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				eventChannel <- Event{Type: "closed_conn"}
				return
			} else {
				fmt.Printf("You have read %s\n", string(buffer[:n]))
				messageChannel <- string(buffer[:n])
			}
		}
		fmt.Println("Exiting from read from TCP loop")
	}()

	// here we are reading from the event channel and writing to the TCP connection
	for !shouldQuit {
		fmt.Println("Running loop for reading from event channel")
		ev := <-eventChannel
		if ev.Type == "quit" || ev.Type == "closed_conn" {
			shouldQuit = true
			return
		}
		fmt.Printf("Sending %s\n", ev.Msg)
		conn.Write([]byte(ev.Msg))
	}
	fmt.Println("Exiting from read from event channel loop")
}

func computeHandshakeKey(uid string) string {
	// Append the shared GUID to the client GUID
	finalGUID := uid + sharedGUID

	// Take sha1 and base64 encode the result
	finalGUIDBytes := sha1.Sum([]byte(finalGUID))
	finalGUIDEncoded := base64.StdEncoding.EncodeToString(finalGUIDBytes[:])

	return finalGUIDEncoded
}

func StartChatServer(addr string) (chan string, chan Event) {
	fmt.Printf("chat server listening on %s\n", addr)
	messageChannel := make(chan string)
	eventChannel := make(chan Event)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		// Check for the existence of Sec-WebSocket-Key Header
		dump, _ := httputil.DumpRequest(r, true)
		fmt.Printf("Request:\n%s", dump)
		clientGuidEncoded := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
		if clientGuidEncoded == "" {
			http.Error(w, "Header missing Sec-WebSocket-Key", http.StatusBadRequest)
			return
		}

		finalGUIDEncoded := computeHandshakeKey(clientGuidEncoded)

		// Then respond with HTTP Upgrade
		w.Header().Add("Sec-WebSocket-Accept", finalGUIDEncoded)
		w.Header().Add("Connection", "Upgrade")
		w.Header().Add("Upgrade", "websocket")
		w.WriteHeader(101)

		// Now we hijack the underlying tcp connection and use it with websocket communication
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		conn, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		go handleChatMessages(conn, messageChannel, eventChannel)
	})

	go http.ListenAndServe(":8080", nil)
	return messageChannel, eventChannel
}

func ConnectToChatServer(host string, port uint16) (chan Message, chan Event, error) {
	tcpAddr := fmt.Sprintf("%s:%d", host, port)
	httpAddr := fmt.Sprintf("http://%s/chat", tcpAddr)

	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Error: could not connect to %s\n", tcpAddr)
		return nil, nil, err
	}

	// Set up the handshake http GET request
	req, err := http.NewRequest("GET", httpAddr, nil)
	if err != nil {
		return nil, nil, err
	}
	clientGuid := uuid.New()
	encodedGuid := base64.StdEncoding.EncodeToString([]byte(clientGuid.String()))
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "websocket")
	req.Header.Add("Sec-Websocket-Key", encodedGuid)
	req.Header.Add("Sec-Websocket-Protocol", "chat")

	// Perform the handshake http GET request. Here we use the tcp connection already created
	// to send the handshake http GET request
	client := http.Client{Transport: &http.Transport{Dial: connDialer{conn}.dial}}
	fmt.Printf("Calling %s\n", req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	dump, _ := httputil.DumpResponse(resp, false)
	fmt.Printf("Handshake Response:\n%s", dump)

	// Verify the Status code
	if resp.StatusCode != 101 {
		return nil, nil, ChatServerConnectionError("handshake status code is not 101")
	}

	// Verify the Upgrade header to be websocket
	if resp.Header.Get("Upgrade") != "websocket" || resp.Header.Get("Connection") != "Upgrade" {
		return nil, nil, ChatServerConnectionError("handshake response is not an upgrade to websocket")
	}

	// Verify the Sec-Websocket-Accept header
	respKey := resp.Header.Get("Sec-Websocket-Accept")
	expectedKey := computeHandshakeKey(encodedGuid)
	if expectedKey != respKey {
		return nil, nil, ChatServerConnectionError("handshake response key is invalid")
	}

	messageChannel := make(chan Message)
	eventChannel := make(chan Event)

	go runChatProtocol(conn, messageChannel, eventChannel)

	return messageChannel, eventChannel, nil
}
