package graphtest

import (
	"github.com/stretchr/testify/suite"
	"linkrus/linkgraph/graph"
)

// SuiteBase defines a re-usable set of graph-related
// test that can be executed against any type that implements
// graph.Graph.
type SuiteBase struct {
	g graph.Graph
	suite.Suite
}
