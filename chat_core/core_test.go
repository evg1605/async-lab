package chat_core_test

import (
	"context"
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
