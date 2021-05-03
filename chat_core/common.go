package chat_core

import (
	"context"
	"errors"
)

type Message struct {
	ID      int64
	Content string
}

var ErrDisconnectedByClient = errors.New("client was disconnected")

type AddMessageCmd struct {
	Content string
	Result  chan *AddMessageResult
}

type AddMessageResult struct {
	Err error
	Msg *Message
}

type NewClientCmd struct {
	CloseCtx  context.Context
	LastMsgID int64
	Count     int
	Result    chan *NewClientResult
}

type NewClientResult struct {
	ClientID         int64
	Messages         <-chan []*Message
	ClosedByServerCh <-chan error
}

type DisconnectClientCmd struct {
	ClientID int64
}

type Storage interface {
	AddMessage(ctx context.Context, content string) (*Message, error)
	GetLastMessages(ctx context.Context, fromID int64, count int) ([]*Message, error)
	GetLastMsgID(ctx context.Context) (int64, error)
}
