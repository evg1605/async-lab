package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

type chatServer struct {
	chat_contracts.UnimplementedChatServer
	mainCtx      context.Context
	chatCommands chan<- interface{}
}

var errGrpcShutdownServer = status.Error(codes.Unavailable, "shutdown server")

func newChatServer(ctx context.Context, chatCommands chan<- interface{}) *chatServer {
	return &chatServer{
		mainCtx:      ctx,
		chatCommands: chatCommands,
	}
}

func (cs *chatServer) SendMessage(ctx context.Context, cmd *chat_contracts.SendMessageCmd) (*chat_contracts.ChatMsg, error) {
	chatCmd, resultCh := chat_core.CreateSendMessageCmd(cmd.UserID, cmd.Content)

	if err := cs.sendCmd(ctx, chatCmd); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cs.mainCtx.Done():
		return nil, cs.mainCtx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		return chatMessageToGrpc(result.Msg), nil

	}
}

func (cs *chatServer) GetMessages(cmd *chat_contracts.GetMessagesCmd, grpcMessages chat_contracts.Chat_GetMessagesServer) error {
	newClientCmd, closeClient, resultCh := chat_core.CreateNewClientCmd(cmd.LastMsgID, int(cmd.LastMessagesCount))

	if err := cs.sendCmd(grpcMessages.Context(), newClientCmd); err != nil {
		return err
	}

	defer closeClient()

	var cl *chat_core.NewClientResult
	select {
	case cl = <-resultCh:
	case <-grpcMessages.Context().Done():
		return grpcMessages.Context().Err()
	case <-cs.mainCtx.Done():
		return cs.mainCtx.Err()
	}

	chatMessagesCtx, cancelChatMessagesCtx := context.WithCancel(context.Background())
	chatMessages, chatClientClosed := messagesWithBuf(chatMessagesCtx, cl.ClosedByServerCh, cl.Messages)
	defer cancelChatMessagesCtx()

	for {
		select {
		case <-cs.mainCtx.Done():
			return errGrpcShutdownServer
		case <-grpcMessages.Context().Done():
			disconnectClientCmd, _ := chat_core.CreateDisconnectClientCmd(cl.ClientID)
			_ = cs.sendCmd(context.Background(), disconnectClientCmd)
			return grpcMessages.Context().Err()
		case err := <-chatClientClosed:
			return status.Error(codes.Aborted, fmt.Sprintf("chat client disconnected: %s", err))
		case messages := <-chatMessages:
			if err := grpcMessages.Send(&chat_contracts.ChatMessages{Messages: chatMessagesToGrpc(messages)}); err != nil {
				return err
			}
		}
	}

}

func (cs *chatServer) sendCmd(ctx context.Context, chatCmd interface{}) error {
	select {
	case cs.chatCommands <- chatCmd:
	case <-ctx.Done():
		return ctx.Err()
	case <-cs.mainCtx.Done():
		return errGrpcShutdownServer
	}
	return nil
}

func messagesWithBuf(
	ctx context.Context,
	clientClosed <-chan error,
	nonBufMessages <-chan []*chat_core.Message) (<-chan []*chat_core.Message, <-chan error) {

	outMessages := make(chan []*chat_core.Message)
	messagesClosed := make(chan error, 1)

	go func() {
		var resErr error
		defer func() {
			messagesClosed <- resErr
		}()

		var out chan<- []*chat_core.Message
		var bufMessages []*chat_core.Message

		for {
			select {
			case <-ctx.Done():
				resErr = ctx.Err()
				return
			case resErr = <-clientClosed:
				if len(bufMessages) > 0 {
					select {
					case <-ctx.Done():
						resErr = ctx.Err()
					case outMessages <- bufMessages:
					}
				}
				return
			case messages := <-nonBufMessages:
				bufMessages = append(bufMessages, messages...)
				out = outMessages
			case out <- bufMessages:
				bufMessages = nil
				out = nil
			}
		}
	}()

	return outMessages, messagesClosed
}

func chatMessagesToGrpc(messages []*chat_core.Message) []*chat_contracts.ChatMsg {
	var res []*chat_contracts.ChatMsg
	for _, msg := range messages {
		res = append(res, chatMessageToGrpc(msg))
	}
	return res
}

func chatMessageToGrpc(msg *chat_core.Message) *chat_contracts.ChatMsg {
	return &chat_contracts.ChatMsg{
		Id:      msg.ID,
		UserID:  msg.UserID,
		Content: msg.Content,
	}
}
