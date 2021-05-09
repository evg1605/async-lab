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
	ctx          context.Context
	chatCommands chan<- interface{}
	chatClosed   <-chan struct{}
}

func newChatServer(ctx context.Context) *chatServer {
	stor := chat_stub_storage.NewStorage()
	chatCommands, chatClosed := chat_core.StartCore(ctx, stor)
	return &chatServer{
		ctx:          ctx,
		chatCommands: chatCommands,
		chatClosed:   chatClosed,
	}
}

func (cs *chatServer) SendMessage(ctx context.Context, cmd *chat_contracts.SendMessageCmd) (*chat_contracts.ChatMsg, error) {
	sendCmd, sendResultCh := chat_core.CreateSendMessageCmd(cmd.UserID, cmd.Content)
	select {
	case cs.chatCommands <- sendCmd:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cs.ctx.Done():
		return nil, cs.ctx.Err()
	}

	select {
	case sendResult := <-sendResultCh:
		if sendResult.Err != nil {
			return nil, sendResult.Err
		}
		return &chat_contracts.ChatMsg{
			Id:      sendResult.Msg.ID,
			UserID:  sendResult.Msg.UserID,
			Content: sendResult.Msg.Content,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cs.ctx.Done():
		return nil, cs.ctx.Err()
	}
}

func (cs *chatServer) GetMessages(cmd *chat_contracts.GetMessagesCmd, messages chat_contracts.Chat_GetMessagesServer) error {
	for i := 0; i < 1000; i++ {
		select {
		case <-cs.ctx.Done():
			log.Printf("chat context error: %v", cs.ctx.Err())
			return cs.ctx.Err()
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
