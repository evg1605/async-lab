package chat_core

import (
	"context"
	"errors"
)

type Message struct {
	ID      int64
	Content string
}

var (
	ErrDisconnectedByClient = errors.New("client was disconnected")
	ErrDisconnectedByServer = errors.New("server disconnect client")
)

type SendMessageResult struct {
	Err error
	Msg *Message
}

type NewClientResult struct {
	ClientID         int64
	Messages         <-chan []*Message
	ClosedByServerCh <-chan error
}

type Storage interface {
	AddMessage(ctx context.Context, content string) (*Message, error)
	GetLastMessages(ctx context.Context, count int) ([]*Message, error)
	GetMessages(ctx context.Context, fromID int64, count int) ([]*Message, error)
	GetLastMsgID(ctx context.Context) (int64, error)
}
