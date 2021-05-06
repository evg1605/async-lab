package chat_stub_storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	stor := NewStorage()
	require.NotNil(t, stor)

	messages, err := stor.GetLastMessages(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, messages)

	messages, err = stor.GetMessages(context.Background(), 0, 10)
	require.NoError(t, err)
	require.Empty(t, messages)

	countMessages := 10
	for i := 0; i < countMessages; i++ {
		content := fmt.Sprintf("msg-%v", i)
		msg, err := stor.AddMessage(context.Background(), content)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, content, msg.Content)
	}

	messages, err = stor.GetLastMessages(context.Background(), 3)
	require.NoError(t, err)
	require.Len(t, messages, 3)

	messages, err = stor.GetLastMessages(context.Background(), countMessages+10)
	require.NoError(t, err)
	require.Len(t, messages, countMessages)

	messages, err = stor.GetMessages(context.Background(), -1, 3)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, "msg-0", messages[0].Content)
	require.Equal(t, "msg-1", messages[1].Content)
	require.Equal(t, "msg-2", messages[2].Content)

	messages, err = stor.GetMessages(context.Background(), 4, 3)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, "msg-3", messages[0].Content)
	require.Equal(t, "msg-4", messages[1].Content)
	require.Equal(t, "msg-5", messages[2].Content)

	messages, err = stor.GetMessages(context.Background(), 8, 30)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, "msg-7", messages[0].Content)
	require.Equal(t, "msg-8", messages[1].Content)
	require.Equal(t, "msg-9", messages[2].Content)
}
