package websocket_test

import (
	"example/zerochat/websocket"
	"testing"
)

func TestParseWSUrlInvalid(t *testing.T) {
	invalidUrls := []string{
		"ws", "ws://", "ws://127.0.0.1", "127.0.0.1:8080",
		"127.0.0.1:8080/chat", "http://127.0.0.1:8080/chat",
		"http://127.0.0.1/chat", "/chat", "chat",
		"ws://127.0.0.1:8080",
	}
	for _, invalidUrl := range invalidUrls {
		wsUrl, err := websocket.ParseWSUrl(invalidUrl)
		if err == nil {
			t.Errorf("ParseWSUrl(%s) should return an error, insted returned %#v", invalidUrl, wsUrl)
		}
	}
}

func TestParseWSUrlValid(t *testing.T) {
	validUrls := []string{
		"ws://127.0.0.1",
		"wss://127.0.0.1:8080/",
		"wss://127.0.0.1/chat",
		"ws://127.0.0.1:8080/chat",
	}
	for _, validUrl := range validUrls {
		_, err := websocket.ParseWSUrl(validUrl)
		if err != nil {
			t.Errorf("ParseWSUrl(%s) should return a valid url structure, insted returned error %#v", validUrl, err.Error())
		}
	}
}
