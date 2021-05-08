package chat_core_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/chat_stub_storage"
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

	sendMessages(t, commands, "msg-1")
	sendMessages(t, commands, "msg-2")

}

func sendMessages(t *testing.T, commands chan<- interface{}, content string) {
	cmd, res := chat_core.CreateSendMessageCmd(content)
	commands <- cmd
	smRes := <-res

	require.NoError(t, smRes.Err)
	require.NotNil(t, smRes.Msg)
	require.Equal(t, content, smRes.Msg.Content)
}
