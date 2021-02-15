package messagetest

import (
	"fmt"
	gc "gopkg.in/check.v1"
	"linkrus/bspgraph/message"
)

type Suite struct {
	q message.Queue
}

func (s *Suite) SetQueue(q message.Queue) {
	s.q = q
}

func (s *Suite) TestSanity(c *gc.C) {
	c.Assert(s.q, gc.NotNil)
}

func (s *Suite) TestEnqueueDequeue(c *gc.C) {

	// Enqueue an item
	for i := 0; i < 10; i++ {
		err := s.q.Enqueue(msg{payload: fmt.Sprint(i)})
		c.Assert(err, gc.IsNil)
	}

	// Assert if there's a pending messages
	c.Assert(s.q.PendingMessages(), gc.Equals, true)

	// Expecting the message to be dequeued in reverse order
	var (
		iter      = s.q.Messages()
		processed int
	)

	for expNext := 9; iter.Next(); expNext-- {
		got := iter.Message().(msg).payload
		c.Assert(got, gc.Equals, fmt.Sprint(expNext))
		processed++
	}

	c.Assert(processed, gc.Equals, 10)
	c.Assert(iter.Error(), gc.IsNil)
}

func (s *Suite) TestDiscard(c *gc.C) {
	// Enqueue the message
	for i := 0; i < 10; i++ {
		c.Assert(s.q.Enqueue(msg{payload: fmt.Sprint(i)}), gc.IsNil)
	}

	// Check if there's pending
	c.Assert(s.q.PendingMessages(), gc.Equals, true)
	// Discard message
	c.Assert(s.q.DiscardMessages(), gc.IsNil)
	// Check if there's pending
	c.Assert(s.q.PendingMessages(), gc.Equals, false)
}

type msg struct {
	payload string
}

func (msg) Type() string { return "msg" }
