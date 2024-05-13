package chatProto

import "fmt"

type ChatServerConnectionError string

func (err ChatServerConnectionError) Error() string {
	return fmt.Sprintf("Connection to Chat Server rejected. Reason: %s\n", string(err))
}
