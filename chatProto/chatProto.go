package chatProto

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/google/uuid"
)

var (
	sharedGUID           = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	writeChannel         = make(chan Message)
	isWriteChannelClosed = false
)

type Message struct {
	Type     string
	Content  string
	Sender   string
	ChatRoom string
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

func parseMsg(b []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(b, &msg)
	return msg, err
}

func runChatProtocol(conn net.Conn, msgHandler func(Message)) {
	defer conn.Close()
	fmt.Printf("Handle chat messages on conn %v\n", conn)

	// here we are reading from TCP connection and sending the message for processing
	go func() {
		buffer := make([]byte, 10000)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Printf("TCP connection Error: %s\n", err)
				msgHandler(Message{Type: "conn_closed", Content: "", Sender: "", ChatRoom: ""})
				break
			} else {
				fmt.Printf("Got TCP message %s\n", string(buffer[:n]))
				req, err := parseMsg(buffer[:n])
				if err != nil {
					fmt.Printf("Error: %s\n", err)
					continue
				}
				msgHandler(req)
			}
		}
		fmt.Printf("CLOSING GOROUTINE READING FROM CONNECTION %v\n", conn)
	}()

	// here we are reading from the write channel and writing to the TCP connection
	for msg := range writeChannel {
		strMsg, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("Error can't send message, failed to marshall into json: %s\n", err)
			continue
		}
		fmt.Printf("Sending message: %s\n", string(strMsg))
		_, tcpErr := conn.Write(strMsg)
		if tcpErr != nil {
			fmt.Printf("TCP send Error: %s\n", err)
			// TODO: check what kind of error it is. Not every one should break
			break
		}
	}
	fmt.Printf("CLOSING GOROUTINE HANDLING CONNECTION %v\n", conn)
}

func computeHandshakeKey(uid string) string {
	// Append the shared GUID to the client GUID
	finalGUID := uid + sharedGUID

	// Take sha1 and base64 encode the result
	finalGUIDBytes := sha1.Sum([]byte(finalGUID))
	finalGUIDEncoded := base64.StdEncoding.EncodeToString(finalGUIDBytes[:])

	return finalGUIDEncoded
}

func StartChatServer(addr string, msgHandler func(Message)) {
	fmt.Printf("chat server listening on %s\n", addr)

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

		go runChatProtocol(conn, msgHandler)
	})

	http.ListenAndServe(addr, nil)
}

func ConnectToChatServer(host string, port uint16, msgHandler func(Message)) error {
	tcpAddr := fmt.Sprintf("%s:%d", host, port)
	httpAddr := fmt.Sprintf("http://%s/chat", tcpAddr)

	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Error: could not connect to %s\n", tcpAddr)
		return err
	}

	// Set up the handshake http GET request
	req, err := http.NewRequest("GET", httpAddr, nil)
	if err != nil {
		return err
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
		return err
	}
	dump, _ := httputil.DumpResponse(resp, false)
	fmt.Printf("Handshake Response:\n%s", dump)

	// Verify the Status code
	if resp.StatusCode != 101 {
		return ChatServerConnectionError("handshake status code is not 101")
	}

	// Verify the Upgrade header to be websocket
	if resp.Header.Get("Upgrade") != "websocket" || resp.Header.Get("Connection") != "Upgrade" {
		return ChatServerConnectionError("handshake response is not an upgrade to websocket")
	}

	// Verify the Sec-Websocket-Accept header
	respKey := resp.Header.Get("Sec-Websocket-Accept")
	expectedKey := computeHandshakeKey(encodedGuid)
	if expectedKey != respKey {
		return ChatServerConnectionError("handshake response key is invalid")
	}

	go runChatProtocol(conn, msgHandler)

	return nil
}

func Quit() {
	if !isWriteChannelClosed {
		close(writeChannel)
		isWriteChannelClosed = true
	}
}

func SendMsg(msg Message) {
	if !isWriteChannelClosed {
		writeChannel <- msg
	} else {
		fmt.Println("Can't send message. The connection was closed")
	}
}
