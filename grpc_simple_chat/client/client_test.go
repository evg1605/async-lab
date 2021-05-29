package client

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
	"github.com/evg1605/async-lab/grpc_simple_chat/server"
)

type client struct {
	receiveLoopWg    sync.WaitGroup
	userID           string
	cancel           context.CancelFunc
	lastMessageID    int64
	sentMessages     []*chat_contracts.ChatMsg
	receivedMessages []*chat_contracts.ChatMsg
	conn             *grpc.ClientConn
	chat             chat_contracts.ChatClient
}

const address = "localhost:8080"

func TestSimpleSendAndReceive(t *testing.T) {
	gracefulStopChat, chatClosed, err := server.StartChatServer(address)
	require.NoError(t, err)
	defer func() {
		gracefulStopChat()
		<-chatClosed
	}()

	cl1 := startNewClient(t, address, "user-1", -1)
	cl1.sendMessage(t, "m1")
	cl1.sendMessage(t, "m2")

	time.Sleep(time.Second * 3)
	cl1.stop()

	require.Len(t, cl1.sentMessages, 2)
	require.Equal(t, "m1", cl1.sentMessages[0].Content)
	require.Equal(t, "m2", cl1.sentMessages[1].Content)

	require.Len(t, cl1.receivedMessages, len(cl1.sentMessages))
	for i, m := range cl1.sentMessages {
		require.Equal(t, m.Id, cl1.receivedMessages[i].Id)
		require.Equal(t, m.Content, cl1.receivedMessages[i].Content)
	}

}

func TestSendAndReceiveWithSecondClient(t *testing.T) {
	gracefulStopChat, chatClosed, err := server.StartChatServer(address)
	require.NoError(t, err)
	defer func() {
		gracefulStopChat()
		<-chatClosed
	}()

	cl1 := startNewClient(t, address, "user-1", -1)
	cl2 := startNewClient(t, address, "user-2", -1)

	cl1.sendMessage(t, "m1")
	cl1.sendMessage(t, "m2")
	cl1.sendMessage(t, "m3")

	time.Sleep(time.Second * 3)
	cl1.stop()
	cl2.stop()

	require.Len(t, cl2.receivedMessages, len(cl1.sentMessages))

	for i, m := range cl1.sentMessages {
		require.Equal(t, m.Id, cl2.receivedMessages[i].Id)
	}

}

func TestSendAndReceiveWithLateSyncSecondClient(t *testing.T) {
	gracefulStopChat, chatClosed, err := server.StartChatServer(address)
	require.NoError(t, err)
	defer func() {
		gracefulStopChat()
		<-chatClosed
	}()

	cl1 := startNewClient(t, address, "user-1", -1)

	cl1.sendMessage(t, "m1")
	cl1.sendMessage(t, "m2")
	cl1.sendMessage(t, "m3")

	cl1.stop()

	cl2 := startNewClient(t, address, "user-2", cl1.sentMessages[0].Id)

	time.Sleep(time.Second * 3)
	cl2.stop()

	require.Len(t, cl2.receivedMessages, len(cl1.sentMessages)-1)

	//for i, m := range cl1.sentMessages {
	//	require.Equal(t, m.Id, cl2.receivedMessages[i].Id)
	//}

}

func startNewClient(t *testing.T, address, userID string, lastMessageID int64) *client {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(address, opts...)
	require.NoError(t, err)

	chat := chat_contracts.NewChatClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	cl := &client{
		lastMessageID: lastMessageID,
		conn:          conn,
		chat:          chat,
		cancel:        cancel,
		userID:        userID,
	}
	cl.receiveLoopWg.Add(1)
	go cl.receiveLoop(ctx)
	return cl
}

func (cl *client) stop() {
	cl.cancel()
	cl.receiveLoopWg.Wait()
}

func (cl *client) receiveLoop(ctx context.Context) {
	defer cl.receiveLoopWg.Done()
	defer func() {
		_ = cl.conn.Close()
	}()

	for ctx.Err() == nil {
		messagesCl, err := cl.chat.GetMessages(ctx, &chat_contracts.GetMessagesCmd{
			LastMsgID:         cl.lastMessageID,
			LastMessagesCount: int32(1000),
		})
		if err != nil {
			log.Printf("error get messages: %s", err)
			select {
			case <-time.After(time.Second * 3):
			case <-ctx.Done():
				return
			}
			continue
		}
		for {
			result, err := messagesCl.Recv()
			if err != nil {
				log.Printf("error recv messages: %s", err)
				break
			}
			log.Printf("messages received: %v", len(result.Messages))
			for _, message := range result.Messages {
				if cl.lastMessageID < message.Id {
					cl.lastMessageID = message.Id
				}
				cl.receivedMessages = append(cl.receivedMessages, message)
				log.Printf("id: %v, %s, %s", message.Id, message.UserID, message.Content)
			}
		}
	}
}

func (cl *client) sendMessage(t *testing.T, content string) {
	message, err := cl.chat.SendMessage(context.Background(), &chat_contracts.SendMessageCmd{
		UserID:  cl.userID,
		Content: content,
	})
	require.NoError(t, err)
	log.Printf("send message ok: %v, %s, %s", message.Id, message.UserID, message.Content)
	cl.sentMessages = append(cl.sentMessages, message)
}
