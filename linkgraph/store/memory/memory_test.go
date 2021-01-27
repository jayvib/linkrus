package memory

import (
	"github.com/stretchr/testify/suite"
	"linkrus/linkgraph/graph/graphtest"
	"testing"
)

type MemoryTestSuite struct {
	graphtest.SuiteBase
}

func (m *MemoryTestSuite) SetupTest() {
	m.SuiteBase.SetGraph(NewInMemoryGraph())
}

func Test(t *testing.T) {
	suite.Run(t, new(MemoryTestSuite))
}
