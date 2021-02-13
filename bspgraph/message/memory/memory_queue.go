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
	mu sync.Mutex
	msgs []message.Message

	latchedMsg message.Message
}

func (i inMemoryQueue) Close() error {
	return nil
}

func (i inMemoryQueue) Enqueue(msg message.Message) error {
	return nil
}

func (i inMemoryQueue) PendingMessages() bool {
	return false
}

func (i inMemoryQueue) DiscardMessages() bool {
	panic("implement me")
}

func (i inMemoryQueue) Messages() message.Iterator {
	panic("implement me")
}
