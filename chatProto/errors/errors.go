package errors

import "fmt"

type ChatServerConnectionError string

func (err ChatServerConnectionError) Error() string {
	return fmt.Sprintf("Connection to Chat Server rejected. Reason: %s\n", string(err))
}

type WSFrameBuildError string

func (err WSFrameBuildError) Error() string {
	return fmt.Sprintf("Error building websocket frame. Reason: %s\n", string(err))
}

type WSFrameReadError string

func (err WSFrameReadError) Error() string {
	return fmt.Sprintf("Error reading websocket frame. Reason: %s\n", string(err))
}
