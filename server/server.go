package main

import "example/zerochat/websocket"

func main() {
	websocket.Listen(":8080")
}
