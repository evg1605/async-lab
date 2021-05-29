package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evg1605/async-lab/chat_core"
)

func TestMessagesWithBufSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	clientClosed := make(chan error)
	nonBufMessages := make(chan []*chat_core.Message)

	bufMessages, bufMessagesClosed := messagesWithBuf(ctx, clientClosed, nonBufMessages)

	for i := 0; i < 10; i++ {
		nonBufMessages <- []*chat_core.Message{{
			ID:      int64(i),
			Content: fmt.Sprintf("msg-%v", i),
			UserID:  "user-1",
		}}
	}

	messages := <-bufMessages
	require.Len(t, messages, 10)

	cancel()
	select {
	case <-bufMessages:
		require.Fail(t, "can not be closed")
	case err := <-bufMessagesClosed:
		require.Error(t, err)
	}
}
