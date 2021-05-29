package server

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/chat_stub_storage"
	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

type chatServer struct {
	chat_contracts.UnimplementedChatServer
	mainCtx      context.Context
	chatCommands chan<- interface{}
}

var errGrpcShutdownServer = status.Error(codes.Unavailable, "shutdown server")

func StartChatServer(address string) (func(), <-chan struct{}, error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	ctxChat, cancelChat := context.WithCancel(context.Background())
	chatCommands, chatClosed := chat_core.StartCore(ctxChat, chat_stub_storage.NewStorage())

	cs := &chatServer{
		mainCtx:      ctxChat,
		chatCommands: chatCommands,
	}
	chat_contracts.RegisterChatServer(grpcServer, cs)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			cancelChat()
		}
	}()

	return func() {
		cancelChat()
		grpcServer.GracefulStop()
	}, chatClosed, nil
}

func (cs *chatServer) SendMessage(ctx context.Context, cmd *chat_contracts.SendMessageCmd) (*chat_contracts.ChatMsg, error) {
	log.Println("start: SendMessage")
	defer func() {
		log.Println("end: SendMessage")
	}()

	chatCmd, resultCh := chat_core.CreateSendMessageCmd(cmd.UserID, cmd.Content)
	if err := cs.sendCmd(ctx, chatCmd); err != nil {
		log.Printf("error send message to chat commands: %s", err)
		return nil, err
	}

	select {
	case <-ctx.Done():
		log.Printf("error wait chat result, grpc context done: %s", ctx.Err())
		return nil, ctx.Err()
	case <-cs.mainCtx.Done():
		log.Printf("error wait chat result, main context done: %s", cs.mainCtx.Err())
		return nil, errGrpcShutdownServer
	case result := <-resultCh:
		if result.Err != nil {
			log.Printf("error chat result: %s", result.Err)
			return nil, status.Errorf(codes.Internal, "chat server error: %v", result.Err)
		}
		log.Printf("send message success: %v, %s, %s", result.Msg.ID, result.Msg.UserID, result.Msg.Content)
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
		return errGrpcShutdownServer
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
			return status.Errorf(codes.Aborted, "chat client disconnected: %s", err)
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
