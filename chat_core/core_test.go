package chat_core

import (
	"context"
	"sync"
)

type stubStorage struct {
	lock     sync.RWMutex
	idGen    int64
	messages []*Message
}

func newStorage() *stubStorage {
	return &stubStorage{
		lock: sync.RWMutex{},
	}
}

func (ss *stubStorage) AddMessage(ctx context.Context, content string) (*Message, error) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.idGen++
	msg := &Message{
		ID:      ss.idGen,
		Content: content,
	}
	ss.messages = append(ss.messages, msg)
	return msg, nil
}

func (ss *stubStorage) GetLastMessages(ctx context.Context, fromID int64, count int) ([]*Message, error) {
	panic("implement me")
}

func (ss *stubStorage) GetLastMsgID(ctx context.Context) (int64, error) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if len(ss.messages) == 0 {
		return -1, nil
	}
	return ss.messages[len(ss.messages)-1].ID, nil
}
