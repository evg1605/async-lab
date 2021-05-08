package chat_stub_storage

import (
	"context"
	"sync"

	"github.com/evg1605/async-lab/chat_core"
)

type stubStorage struct {
	lock     sync.RWMutex
	idGen    int64
	messages []*chat_core.Message
}

func NewStorage() *stubStorage {
	return &stubStorage{
		lock: sync.RWMutex{},
	}
}

func (ss *stubStorage) AddMessage(ctx context.Context, userID, content string) (*chat_core.Message, error) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.idGen++
	msg := &chat_core.Message{
		ID:      ss.idGen,
		Content: content,
		UserID:  userID,
	}
	ss.messages = append(ss.messages, msg)
	return msg, nil
}

func (ss *stubStorage) GetLastMessages(ctx context.Context, count int) ([]*chat_core.Message, error) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if len(ss.messages) < count {
		return ss.messages[:], nil
	}

	return ss.messages[len(ss.messages)-count:], nil
}

func (ss *stubStorage) GetMessages(ctx context.Context, fromID int64, count int) ([]*chat_core.Message, error) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	for i, msg := range ss.messages {
		if msg.ID >= fromID {
			messages := ss.messages[i:]
			if len(messages) < count {
				return messages, nil
			}
			return messages[:count], nil
		}
	}
	return nil, nil
}

func (ss *stubStorage) GetLastMsgID(ctx context.Context) (int64, error) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if len(ss.messages) == 0 {
		return -1, nil
	}
	return ss.messages[len(ss.messages)-1].ID, nil
}
