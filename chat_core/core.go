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
}

func StartCore(ctx context.Context, stor Storage) chan<- interface{} {
	csrv := &srv{
		clients: make(map[int64]*client),
		cmdsCh:  make(chan interface{}),
		stor:    stor,
	}
	go csrv.mainLoop(ctx)
	return csrv.cmdsCh
}

func (csrv *srv) mainLoop(ctx context.Context) {
	for {
		select {
		case cmd := <-csrv.cmdsCh:
			switch cmd := cmd.(type) {
			case *AddMessageCmd:
				csrv.processAddMessage(ctx, cmd)
			case *NewClientCmd:
				csrv.processNewClient(ctx, cmd)
			case *clientSynchronizedCmd:
				csrv.processClientSynchronized(ctx, cmd)
			}
		}
	}
}

func (csrv *srv) disconnectClient(ctx context.Context, cl *client, disconnectErr error) {
	delete(csrv.clients, cl.id)
	close(cl.messagesCh)
	cl.closedByServerCh <- disconnectErr
}

func (csrv *srv) syncClient(ctx context.Context, clientID, fromID int64, count int) {
	if fromID < 0 {
		messages, err := csrv.stor.GetLastMessages(ctx, fromID, count)
		csrv.cmdsCh <- clientSynchronizedCmd{
			clientID: clientID,
			err:      err,
			messages: messages,
		}
		return
	}

	lastMsgID, err := csrv.stor.GetLastMsgID(ctx)
	if err != nil {
		csrv.cmdsCh <- clientSynchronizedCmd{
			clientID: clientID,
			err:      err,
		}
		return
	}
	var allMessages []*Message
	for {
		messages, err := csrv.stor.GetLastMessages(ctx, fromID, syncMessagesCount)
		if err != nil {
			csrv.cmdsCh <- clientSynchronizedCmd{
				clientID: clientID,
				err:      err,
			}
			return
		}
		allMessages = append(allMessages, messages...)
		if len(messages) == 0 || messages[len(messages)-1].ID >= lastMsgID {
			csrv.cmdsCh <- clientSynchronizedCmd{
				clientID: clientID,
				messages: allMessages,
			}
			return
		}
		fromID = messages[len(messages)-1].ID + 1
	}
}

func (csrv *srv) processNewClient(ctx context.Context, cmd *NewClientCmd) {
	csrv.clientIDGen++
	cl := &client{
		id:                csrv.clientIDGen,
		messagesCh:        make(chan []*Message),
		isSyncDone:        false,
		closedByClientCtx: cmd.CloseCtx,
		closedByServerCh:  make(chan error, 1),
	}
	csrv.clients[cl.id] = cl
	cmd.Result <- &NewClientResult{
		ClientID:         cl.id,
		Messages:         cl.messagesCh,
		ClosedByServerCh: cl.closedByServerCh,
	}
	go csrv.syncClient(ctx, cl.id, cmd.LastMsgID, cmd.Count)
}

func (csrv *srv) processAddMessage(ctx context.Context, cmd *AddMessageCmd) {
	msg, err := csrv.stor.AddMessage(ctx, cmd.Content)
	if err != nil {
		cmd.Result <- &AddMessageResult{Err: err}
		return
	}
	var badClients []*client
	for _, cl := range csrv.clients {
		if !cl.isSyncDone {
			cl.lastMessages = append(cl.lastMessages, msg)
			continue
		}
		select {
		case cl.messagesCh <- []*Message{msg}:
		case <-cl.closedByClientCtx.Done():
			badClients = append(badClients, cl)
		case <-ctx.Done():
			return
		}
	}
	for _, cl := range badClients {
		csrv.disconnectClient(ctx, cl, ErrDisconnectedByClient)
	}
}

func (csrv *srv) processClientSynchronized(ctx context.Context, cmd *clientSynchronizedCmd) {
	cl, ok := csrv.clients[cmd.clientID]
	if !ok {
		return
	}
	if cmd.err != nil {
		csrv.disconnectClient(ctx, cl, cmd.err)
		return
	}

	var messages = cmd.messages
	if len(messages) > 0 {
		messages = append(getMessagesAfter(cl.lastMessages, messages[len(messages)-1].ID))
	} else {
		messages = cl.lastMessages
	}
	cl.isSyncDone = true
	cl.lastMessages = nil
	select {
	case cl.messagesCh <- messages:
	case <-cl.closedByClientCtx.Done():
		csrv.disconnectClient(ctx, cl, ErrDisconnectedByClient)
	case <-ctx.Done():
		return
	}
}

func (csrv *srv) processCloseClient(ctx context.Context, cmd *DisconnectClientCmd) {
	cl, ok := csrv.clients[cmd.ClientID]
	if !ok {
		return
	}
	csrv.disconnectClient(ctx, cl, ErrDisconnectedByClient)
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
