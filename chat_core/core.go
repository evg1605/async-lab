package chat_core

import (
	"context"
)

const syncMessagesCount = 10

type clientSynchronizedCmd struct {
	clientID int64
	err      error
	messages []*Message
}

type client struct {
	id                int64
	isSyncDone        bool
	lastMessages      []*Message
	closedByClientCtx context.Context
	closedByServerCh  chan error
	messagesCh        chan []*Message
}

type srv struct {
	clientIDGen int64
	clients     map[int64]*client
	cmdsCh      chan interface{}
	stor        Storage

	closedCh chan struct{}
}

func StartCore(ctx context.Context, stor Storage) (commands chan<- interface{}, chatClosed <-chan struct{}) {
	csrv := &srv{
		clients:  make(map[int64]*client),
		cmdsCh:   make(chan interface{}),
		stor:     stor,
		closedCh: make(chan struct{}),
	}
	go csrv.mainLoop(ctx)
	return csrv.cmdsCh, csrv.closedCh
}

func (csrv *srv) mainLoop(ctx context.Context) {

	defer func() {
		for _, cl := range csrv.clients {
			csrv.disconnectClient(cl, ErrDisconnectedByServer)
		}
		close(csrv.closedCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case cmd := <-csrv.cmdsCh:
			switch cmd := cmd.(type) {
			case *sendMessageCmd:
				csrv.processSendMessage(ctx, cmd)
			case *newClientCmd:
				csrv.processNewClient(ctx, cmd)
			case *clientSynchronizedCmd:
				csrv.processClientSynchronized(ctx, cmd)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (csrv *srv) disconnectClient(cl *client, disconnectErr error) {
	delete(csrv.clients, cl.id)
	close(cl.messagesCh)
	cl.closedByServerCh <- disconnectErr
}

func (csrv *srv) syncClient(ctx context.Context, clientID, fromID int64, lastMessagesCount int) {

	sendResult := func(cmd *clientSynchronizedCmd) {
		select {
		case csrv.cmdsCh <- cmd:
		case <-ctx.Done():
		}
	}

	if fromID < 0 {
		if lastMessagesCount <= 0 {
			sendResult(&clientSynchronizedCmd{
				clientID: clientID,
			})
			return
		}
		messages, err := csrv.stor.GetLastMessages(ctx, lastMessagesCount)
		sendResult(&clientSynchronizedCmd{
			clientID: clientID,
			err:      err,
			messages: messages,
		})
		return
	}

	lastMsgID, err := csrv.stor.GetLastMsgID(ctx)
	if err != nil {
		sendResult(&clientSynchronizedCmd{
			clientID: clientID,
			err:      err,
		})
		return
	}
	var allMessages []*Message
	for {
		messages, err := csrv.stor.GetMessages(ctx, fromID, syncMessagesCount)
		if err != nil {
			sendResult(&clientSynchronizedCmd{
				clientID: clientID,
				err:      err,
			})
			return
		}
		allMessages = append(allMessages, messages...)
		if len(messages) == 0 || messages[len(messages)-1].ID >= lastMsgID {
			sendResult(&clientSynchronizedCmd{
				clientID: clientID,
				messages: allMessages,
			})
			return
		}
		fromID = messages[len(messages)-1].ID + 1
	}
}

func (csrv *srv) processNewClient(ctx context.Context, cmd *newClientCmd) {
	csrv.clientIDGen++
	cl := &client{
		id:                csrv.clientIDGen,
		messagesCh:        make(chan []*Message),
		isSyncDone:        false,
		closedByClientCtx: cmd.closeCtx,
		closedByServerCh:  make(chan error, 1),
	}
	csrv.clients[cl.id] = cl
	cmd.result <- &NewClientResult{
		ClientID:         cl.id,
		Messages:         cl.messagesCh,
		ClosedByServerCh: cl.closedByServerCh,
	}
	go csrv.syncClient(ctx, cl.id, cmd.lastMsgID, cmd.lastMessagesCount)
}

func (csrv *srv) processSendMessage(ctx context.Context, cmd *sendMessageCmd) {
	msg, err := csrv.stor.AddMessage(ctx, cmd.content)
	if err != nil {
		cmd.result <- &SendMessageResult{Err: err}
		return
	}

	cmd.result <- &SendMessageResult{Msg: msg}

	for _, cl := range csrv.clients {
		if !cl.isSyncDone {
			cl.lastMessages = append(cl.lastMessages, msg)
			continue
		}
		select {
		case cl.messagesCh <- []*Message{msg}:
		case <-cl.closedByClientCtx.Done():
			csrv.disconnectClient(cl, ErrDisconnectedByClient)
		case <-ctx.Done():
			return
		}
	}
}

func (csrv *srv) processClientSynchronized(ctx context.Context, cmd *clientSynchronizedCmd) {
	cl, ok := csrv.clients[cmd.clientID]
	if !ok {
		return
	}
	if cmd.err != nil {
		csrv.disconnectClient(cl, cmd.err)
		return
	}

	var messages = cmd.messages
	if len(messages) > 0 {
		messages = append(messages, getMessagesAfter(cl.lastMessages, messages[len(messages)-1].ID)...)
	} else {
		messages = cl.lastMessages
	}
	cl.isSyncDone = true
	cl.lastMessages = nil
	if len(messages) == 0 {
		return
	}

	select {
	case cl.messagesCh <- messages:
	case <-cl.closedByClientCtx.Done():
		csrv.disconnectClient(cl, ErrDisconnectedByClient)
	case <-ctx.Done():
		return
	}
}

func (csrv *srv) processCloseClient(ctx context.Context, cmd *disconnectClientCmd) {
	cl, ok := csrv.clients[cmd.clientID]
	if !ok {
		return
	}
	csrv.disconnectClient(cl, ErrDisconnectedByClient)
}

func getMessagesAfter(messages []*Message, afterID int64) []*Message {
	// todo binary search more effective
	for i, msg := range messages {
		if msg.ID > afterID {
			return messages[i:]
		}
	}
	return nil
}
