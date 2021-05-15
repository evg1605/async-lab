package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/chat_stub_storage"
	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

type chatServer struct {
	chat_contracts.UnimplementedChatServer
	mainCtx          context.Context
	chatCommands     chan<- interface{}
	chatServerClosed <-chan struct{}
}

func newChatServer(ctx context.Context) *chatServer {
	stor := chat_stub_storage.NewStorage()
	chatCommands, chatServerClosed := chat_core.StartCore(ctx, stor)
	return &chatServer{
		mainCtx:          ctx,
		chatCommands:     chatCommands,
		chatServerClosed: chatServerClosed,
	}
}

func (cs *chatServer) SendMessage(ctx context.Context, cmd *chat_contracts.SendMessageCmd) (*chat_contracts.ChatMsg, error) {
	chatCmd, resultCh := chat_core.CreateSendMessageCmd(cmd.UserID, cmd.Content)

	if err := cs.sendCmd(ctx, chatCmd); err != nil {
		return nil, err
	}

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		return &chat_contracts.ChatMsg{
			Id:      result.Msg.ID,
			UserID:  result.Msg.UserID,
			Content: result.Msg.Content,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cs.mainCtx.Done():
		return nil, cs.mainCtx.Err()
	}
}

func (cs *chatServer) GetMessages(cmd *chat_contracts.GetMessagesCmd, messages chat_contracts.Chat_GetMessagesServer) error {
	chatCmd, closeClient, resultCh := chat_core.CreateNewClientCmd(cmd.LastMsgID, int(cmd.LastMessagesCount))

	if err := cs.sendCmd(messages.Context(), chatCmd); err != nil {
		return err
	}

	defer closeClient()

	var cl *chat_core.NewClientResult
	select {
	case cl = <-resultCh:
	case <-messages.Context().Done():
		return messages.Context().Err()
	case <-cs.mainCtx.Done():
		return cs.mainCtx.Err()
	}

	_ = cl
	for i := 0; i < 1000; i++ {
		select {
		case <-cs.mainCtx.Done():
			log.Printf("chat context error: %v", cs.mainCtx.Err())
			return cs.mainCtx.Err()
		case <-messages.Context().Done():
			log.Printf("stream context error: %v", messages.Context().Err())
			return messages.Context().Err()
		case <-time.Tick(time.Second * 1):
		}

		if err := messages.Send(&chat_contracts.ChatMessages{Messages: []*chat_contracts.ChatMsg{{
			Id:      int64(i),
			UserID:  "user-1",
			Content: fmt.Sprintf("message-%v", i),
		}}}); err != nil {
			log.Printf("send message to client error: %v", err)
			return err
		}
		log.Printf("send message %v success", i)
	}
	return nil
}

func (cs *chatServer) sendCmd(ctx context.Context, chatCmd interface{}) error {
	select {
	case cs.chatCommands <- chatCmd:
	case <-ctx.Done():
		return ctx.Err()
	case <-cs.mainCtx.Done():
		return cs.mainCtx.Err()
	}
	return nil
}

func messagesWithBuf(ctx context.Context,
	serverClosed <-chan struct{},
	clientClosed <-chan error,
	nonBufMessages <-chan []*chat_core.Message) <-chan []*chat_core.Message {
	outMessages := make(chan []*chat_core.Message)

	go func() {
		defer close(outMessages)

		var out chan<- []*chat_core.Message
		var bufMessages []*chat_core.Message

		for {
			select {
			case <-ctx.Done():
				return
			case <-serverClosed:
				return
			case <-clientClosed:
				return
			case messages, ok := <-nonBufMessages:
				if !ok {
					return
				}
				bufMessages = append(bufMessages, messages...)
				out = outMessages
			case out <- bufMessages:
				bufMessages = nil
				out = nil
			}
		}
	}()

	return outMessages
}
