package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/evg1605/async-lab/chat_core"
)

func TestMessagesWithBufSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serverClosed := make(chan struct{})
	clientClosed := make(chan error)
	nonBufMessages := make(chan []*chat_core.Message)

	bufMessages := messagesWithBuf(ctx, serverClosed, clientClosed, nonBufMessages)

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
	_, ok := <-bufMessages
	require.False(t, ok)
}
