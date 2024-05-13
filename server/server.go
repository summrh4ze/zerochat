package main

import ws "example/zerochat/websocket"

func main() {
	ws.Listen(":8080")
}
