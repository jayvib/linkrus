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

}

type msg struct {
	payload string
}

func (msg) Type() string { return "msg" }


