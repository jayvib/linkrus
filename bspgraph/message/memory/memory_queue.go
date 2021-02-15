package memory

import (
	"linkrus/bspgraph/message"
	"sync"
)

func New() message.Queue {
	return &inMemoryQueue{}
}

var _ message.Queue = (*inMemoryQueue)(nil)

type inMemoryQueue struct {
	mu   sync.RWMutex
	msgs []message.Message

	latchedMsg message.Message
}

func (i *inMemoryQueue) Close() error {
	return nil
}

func (i *inMemoryQueue) Enqueue(msg message.Message) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.msgs = append(i.msgs, msg)

	return nil
}

func (i *inMemoryQueue) PendingMessages() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.msgs) > 0
}

func (i *inMemoryQueue) DiscardMessages() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.msgs = i.msgs[:0]
	i.latchedMsg = nil
	return nil
}

func (i *inMemoryQueue) Messages() message.Iterator {
	return i
}

func (i *inMemoryQueue) Next() bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	qLen := len(i.msgs)
	if qLen == 0 {
		return false
	}

	i.latchedMsg = i.msgs[qLen-1]
	i.msgs = i.msgs[:qLen-1]

	return true
}

func (i *inMemoryQueue) Error() error {
	return nil
}

func (i *inMemoryQueue) Message() message.Message {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.latchedMsg
}
