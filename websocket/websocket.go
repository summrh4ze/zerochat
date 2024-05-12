package websocket

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
)

type WSConn struct {
	conn net.Conn
}

type WSConnError struct {
	host string
	port string
}

type WSUrlFormatError string

func (err WSConnError) Error() string {
	return fmt.Sprintf("Connection to %s:%s rejected\n", err.host, err.port)
}

func (err WSUrlFormatError) Error() string {
	return fmt.Sprintf("%s is not a valid websocket url\n", string(err))
}

func handleWebsocketConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("NOW WE SPEAK WEBSOCKET")
}

func Listen(addr string) {
	fmt.Printf("websocket server listening on %s\n", addr)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		// First respond with HTTP Upgrade
		dump, _ := httputil.DumpRequest(r, true)
		fmt.Printf("Request:\n%s", dump)
		w.Header().Add("Sec-WebSocket-Accept", "blablabla-123")
		w.Header().Add("Connection", "Upgrade")
		w.Header().Add("Upgrade", "websocket")
		w.WriteHeader(101)

		// Now we hijack the underlying tcp connection and use it for websocket communication
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
		go handleWebsocketConnection(conn)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ParseWSUrl(addr string) (*url.URL, error) {
	wsUrl, err := url.ParseRequestURI(addr)
	if err != nil {
		return &url.URL{}, err
	}

	hasScheme, _ := regexp.MatchString("ws|wss", wsUrl.Scheme)
	hasHost := wsUrl.Host != ""
	hasPath := wsUrl.Path != ""

	if !hasScheme || !hasHost || !hasPath {
		return &url.URL{}, WSUrlFormatError(addr)
	}

	return wsUrl, nil
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

func Connect(addr string) (WSConn, error) {
	wsUrl, err := ParseWSUrl(addr)
	if err != nil {
		return WSConn{}, err
	}

	tcpAddr := wsUrl.Hostname() + ":" + wsUrl.Port()
	wsUrl.Scheme = "http"
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Error: could not connect to %s\n", addr)
		return WSConn{}, err
	}

	req, err := http.NewRequest("GET", wsUrl.String(), nil)
	if err != nil {
		return WSConn{}, err
	}
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "websocket")
	req.Header.Add("Sec-Websocket-Key", "blablabla")
	req.Header.Add("Sec-Websocket-Protocol", "chat")
	client := http.Client{Transport: &http.Transport{Dial: connDialer{conn}.Dial}}
	fmt.Printf("Calling %s\n", req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return WSConn{}, err
	}
	dump, _ := httputil.DumpResponse(resp, false)
	fmt.Printf("Handshake Response:\n%s", dump)

	return WSConn{conn}, nil
}

func (wsConn WSConn) Close() {
	fmt.Printf("Closing connection\n")
	wsConn.conn.Close()
}
