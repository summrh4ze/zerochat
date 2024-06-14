package chatProto

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"example/zerochat/chatProto/errors"
	"example/zerochat/chatProto/websockets"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"slices"
	"strings"
	"sync"
)

var (
	sharedGUID        = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	registeredClients = make(map[string]*Client)
	mutex             sync.Mutex
)

type Client struct {
	conn                 net.Conn
	writeChannel         chan Message
	name                 string
	id                   string
	isWriteChannelClosed bool
}

type Message struct {
	Type       string
	Content    string
	Sender     string
	Receipient string
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

func runChatProtocol(client *Client, msgHandler func(Message)) {
	defer client.conn.Close()
	// notify all users that a new user connected
	go notifyConnect(client.name, client.id)
	fmt.Printf("Handle chat messages on conn %v\n", client.conn)

	// here we are reading from TCP connection and sending the message for processing
	go func() {
		//buffer := make([]byte, 10000)
		for {
			payload, err := websockets.ReadMessage(client.conn)
			if err != nil {
				fmt.Printf("TCP connection Error: %s\n", err)
				msgHandler(Message{Type: "conn_closed", Content: "", Sender: ""})
				break
			} else {
				fmt.Printf("Got TCP message %s\n", string(payload))
				req, err := parseMsg(payload)
				if err != nil {
					fmt.Printf("Error: %s\n", err)
					continue
				}
				msgHandler(req)
			}
		}
		fmt.Printf("CLOSING GOROUTINE READING FROM CONNECTION %v\n", client.conn)
		if !client.isWriteChannelClosed {
			close(client.writeChannel)
		}
	}()

	// here we are reading from the write channel and writing to the TCP connection
	for msg := range client.writeChannel {
		encMsgBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("Error can't send message, failed to marshall into json: %s\n", err)
			continue
		}
		fmt.Printf("Sending message: %s\n", []byte(encMsgBytes))
		tcpErr := websockets.CreateMessage(client.conn, encMsgBytes, false)
		if tcpErr != nil {
			fmt.Printf("TCP send Error: %s\n", err)
			// TODO: check what kind of error it is. Not every one should break
			break
		}
	}
	fmt.Printf("CLOSING GOROUTINE HANDLING CONNECTION %v\n", client.conn)
	mutex.Lock()
	defer mutex.Unlock()
	delete(registeredClients, client.id)
	// notify all users that a user disconnected
	go notifyDisconnect(client.id)
}

func notifyDisconnect(clientId string) {
	for _, rc := range registeredClients {
		if !rc.isWriteChannelClosed {
			rc.writeChannel <- Message{Type: CMD_USER_DISCONNECTED, Content: clientId}
		}
	}
}

func notifyConnect(clientName string, clientId string) {
	for id, rc := range registeredClients {
		if id != clientId && !rc.isWriteChannelClosed {
			rc.writeChannel <- Message{
				Type:    CMD_USER_CONNECTED,
				Content: fmt.Sprintf("%s,%s", clientName, clientId),
			}
		}
	}
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
		name := r.URL.Query().Get("name")
		id := r.URL.Query().Get("id")
		fmt.Printf("User connected: %s - %s\n", name, id)

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

		writeChannel := make(chan Message)
		client := Client{
			conn:         conn,
			writeChannel: writeChannel,
			name:         name,
			id:           id,
		}
		mutex.Lock()
		defer mutex.Unlock()
		registeredClients[id] = &client

		go runChatProtocol(&client, msgHandler)
	})

	http.ListenAndServe(addr, nil)
}

func ConnectToChatServer(host string, port string, name string, id string, msgHandler func(Message)) error {
	tcpAddr := fmt.Sprintf("%s:%s", host, port)
	httpAddr := fmt.Sprintf("http://%s/chat?name=%s&id=%s", tcpAddr, name, id)

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
	nonce := make([]byte, 16)
	_, randerr := rand.Read(nonce)
	if randerr != nil {
		return randerr
	}

	encodedGuid := base64.StdEncoding.EncodeToString(nonce)
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
		return errors.ChatServerConnectionError("handshake status code is not 101")
	}

	// Verify the Upgrade header to be websocket
	if resp.Header.Get("Upgrade") != "websocket" || resp.Header.Get("Connection") != "Upgrade" {
		return errors.ChatServerConnectionError("handshake response is not an upgrade to websocket")
	}

	// Verify the Sec-Websocket-Accept header
	respKey := resp.Header.Get("Sec-Websocket-Accept")
	expectedKey := computeHandshakeKey(encodedGuid)
	if expectedKey != respKey {
		return errors.ChatServerConnectionError("handshake response key is invalid")
	}

	writeChannel := make(chan Message)

	user := Client{
		conn:         conn,
		writeChannel: writeChannel,
		name:         name,
		id:           id,
	}
	mutex.Lock()
	defer mutex.Unlock()
	registeredClients[id] = &user

	go runChatProtocol(&user, msgHandler)

	return nil
}

func ClientQuit(id string) {
	mutex.Lock()
	defer mutex.Unlock()
	if len(registeredClients) != 1 {
		panic("ERROR: client should have been registered in the protocol")
	}
	c := registeredClients[id]
	if !c.isWriteChannelClosed {
		close(c.writeChannel)
		c.isWriteChannelClosed = true
	}
}

func ClientSendMsg(msg Message, id string) {
	mutex.Lock()
	defer mutex.Unlock()
	if len(registeredClients) != 1 {
		panic("ERROR: client should have been registered in the protocol")
	}
	c := registeredClients[id]
	if !c.isWriteChannelClosed {
		c.writeChannel <- msg
	} else {
		fmt.Println("Can't send message. The connection was closed")
	}
}

func verifyClient(user string) *Client {
	userDetails := strings.Split(user, ",")
	if len(userDetails) != 2 {
		fmt.Println("GET USERS: Can't determine the user that sent the request")
		return nil
	}
	mutex.Lock()
	defer mutex.Unlock()
	if c, ok := registeredClients[userDetails[1]]; !ok {
		fmt.Printf("GET USERS: %s,%s not registered\n", userDetails[0], userDetails[1])
		return nil
	} else {
		return c
	}
}

func GetUsers(msg Message) {
	if c := verifyClient(msg.Sender); c == nil {
		return
	} else {
		mutex.Lock()
		defer mutex.Unlock()
		var resp Message
		resp.Sender = msg.Sender
		resp.Type = CMD_GET_USERS_RESPONSE
		fmt.Printf("there are %d clients registered\n", len(registeredClients))
		users := make([]string, 0, len(registeredClients))
		for _, v := range registeredClients {
			if v.id != c.id {
				res := v.name + "," + v.id
				fmt.Printf("%s\n", res)
				users = append(users, res)
				fmt.Printf("users %v\n", users)
			}
		}
		// sort users alphabetically
		slices.Sort(users)
		resp.Content = strings.Join(users, "\n")
		if !c.isWriteChannelClosed {
			c.writeChannel <- resp
		} else {
			fmt.Println("Could not send GET_USERS response because the client disconnected")
		}

	}
}

func SendMessage(msg Message) {
	if s := verifyClient(msg.Sender); s == nil {
		return
	} else {
		if r := verifyClient(msg.Receipient); r == nil {
			fmt.Printf("Error: the receiver of the message does not exist. Aborting SEND_MSG operation.")
			return
		} else {
			if !r.isWriteChannelClosed {
				r.writeChannel <- msg
			} else {
				fmt.Println("Could not send message because the receipient disconnected")
			}
		}
	}
}
