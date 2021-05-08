package chat_core

import "context"

type newClientCmd struct {
	closeCtx          context.Context
	lastMsgID         int64
	lastMessagesCount int
	result            chan *NewClientResult
}

func CreateNewClientCmd(lastMsgID int64, lastMessagesCount int) (cmd interface{}, closeClient context.CancelFunc, result <-chan *NewClientResult) {
	ctx, cancel := context.WithCancel(context.Background())

	ncCmd := &newClientCmd{
		closeCtx:          ctx,
		lastMsgID:         lastMsgID,
		lastMessagesCount: lastMessagesCount,
		result:            make(chan *NewClientResult, 1),
	}

	return ncCmd, cancel, ncCmd.result
}

type sendMessageCmd struct {
	content string
	result  chan *SendMessageResult
}

func CreateSendMessageCmd(content string) (cmd interface{}, result <-chan *SendMessageResult) {
	smCmd := &sendMessageCmd{
		content: content,
		result:  make(chan *SendMessageResult, 1),
	}

	return smCmd, smCmd.result
}

type disconnectClientCmd struct {
	clientID int64
	result   chan struct{}
}

func CreateDisconnectClientCmd(clientID int64) (cmd interface{}, result <-chan struct{}) {
	dcCmd := &disconnectClientCmd{
		clientID: clientID,
		result:   make(chan struct{}, 1),
	}
	return dcCmd, dcCmd.result
}
