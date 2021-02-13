package memory

import (
	gc "gopkg.in/check.v1"
	"linkrus/bspgraph/message"
	"linkrus/bspgraph/message/messagetest"
	"testing"
)

var _ = gc.Suite(new(MemoryQueueTestSuite))

type MemoryQueueTestSuite struct {
	messagetest.Suite
	q message.Queue
}

func (s *MemoryQueueTestSuite) SetUpTest(_ *gc.C) {
	s.q = New()
	s.SetQueue(s.q)
}

func (s *MemoryQueueTestSuite) TearDownTest(c *gc.C) {
	c.Assert(s.q.Close(), gc.IsNil)
}

func Test(t *testing.T) {
	gc.TestingT(t)
}

