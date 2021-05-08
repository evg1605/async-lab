package chat_core_test

import (
	"context"
	"testing"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/chat_stub_storage"
	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	stor := chat_stub_storage.NewStorage()

	ctx, cancel := context.WithCancel(context.Background())

	_, closed := chat_core.StartCore(ctx, stor)

	cancel()

	<-closed
}

func TestNewClient(t *testing.T) {
	commands, _ := chat_core.StartCore(
		context.Background(),
		chat_stub_storage.NewStorage())

	cmd, _, res := chat_core.CreateNewClientCmd(-1, 0)
	commands <- cmd

	ncr := <-res
	require.NotNil(t, ncr)
}

func TestSendMessages(t *testing.T) {
	stor := chat_stub_storage.NewStorage()
	commands, _ := chat_core.StartCore(context.Background(), stor)

	sendMessage(t, commands, "msg-1")
	sendMessage(t, commands, "msg-2")

	messages, err := stor.GetLastMessages(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "msg-1", messages[0].Content)
	require.Equal(t, "msg-2", messages[1].Content)
}

func TestListenMessages(t *testing.T) {
	stor := chat_stub_storage.NewStorage()
	commands, _ := chat_core.StartCore(context.Background(), stor)
	cmd, _, result := chat_core.CreateNewClientCmd(-1, 0)
	commands <- cmd
	clRes := <-result

	sendMessage(t, commands, "msg-1")

	allMessagesCh := receiveMessages(clRes.Messages, 2)

	sendMessage(t, commands, "msg-2")

	allMessages := <-allMessagesCh

	require.Equal(t, "msg-1", allMessages[0].Content)
	require.Equal(t, "msg-2", allMessages[1].Content)
}

func TestListenSyncedLastMessages(t *testing.T) {
	stor := chat_stub_storage.NewStorage()
	commands, _ := chat_core.StartCore(context.Background(), stor)

	sendMessage(t, commands, "msg-1")
	sendMessage(t, commands, "msg-2")
	sendMessage(t, commands, "msg-3")

	cmd, _, result := chat_core.CreateNewClientCmd(-1, 2)
	commands <- cmd
	clRes := <-result

	allMessagesCh := receiveMessages(clRes.Messages, 4)

	sendMessage(t, commands, "msg-4")
	sendMessage(t, commands, "msg-5")

	allMessages := <-allMessagesCh

	require.Equal(t, "msg-2", allMessages[0].Content)
	require.Equal(t, "msg-3", allMessages[1].Content)
	require.Equal(t, "msg-4", allMessages[2].Content)
	require.Equal(t, "msg-5", allMessages[3].Content)
}

func TestListenSyncedMessagesFrom(t *testing.T) {
	stor := chat_stub_storage.NewStorage()
	commands, _ := chat_core.StartCore(context.Background(), stor)

	sendMessage(t, commands, "msg-1")
	lastReadMsg := sendMessage(t, commands, "msg-2")
	sendMessage(t, commands, "msg-3")

	cmd, _, result := chat_core.CreateNewClientCmd(lastReadMsg.ID+1, 0)
	commands <- cmd
	clRes := <-result

	allMessagesCh := receiveMessages(clRes.Messages, 3)

	sendMessage(t, commands, "msg-4")
	sendMessage(t, commands, "msg-5")

	allMessages := <-allMessagesCh

	require.Equal(t, "msg-3", allMessages[0].Content)
	require.Equal(t, "msg-4", allMessages[1].Content)
	require.Equal(t, "msg-5", allMessages[2].Content)
}

func TestDisconnectClient(t *testing.T) {
	stor := chat_stub_storage.NewStorage()
	commands, _ := chat_core.StartCore(context.Background(), stor)

	cmd, _, result := chat_core.CreateNewClientCmd(-1, 0)
	commands <- cmd
	clRes := <-result

	dcCmd, dcRes := chat_core.CreateDisconnectClientCmd(clRes.ClientID)

	commands <- dcCmd
	<-dcRes

	<-clRes.ClosedByServerCh
}

func receiveMessages(messages <-chan []*chat_core.Message, count int) <-chan []*chat_core.Message {
	res := make(chan []*chat_core.Message, 1)
	go func() {
		var allMessages []*chat_core.Message
		for len(allMessages) < count {
			messages := <-messages
			allMessages = append(allMessages, messages...)
		}
		res <- allMessages
	}()
	return res
}

func sendMessage(t *testing.T, commands chan<- interface{}, content string) *chat_core.Message {
	cmd, res := chat_core.CreateSendMessageCmd("userID", content)
	commands <- cmd
	smRes := <-res

	require.NoError(t, smRes.Err)
	require.NotNil(t, smRes.Msg)
	require.Equal(t, content, smRes.Msg.Content)

	return smRes.Msg
}
