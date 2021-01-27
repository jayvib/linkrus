package graphtest

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
	"linkrus/linkgraph/graph"
	"time"
)

// SuiteBase defines a re-usable set of graph-related
// test that can be executed against any type that implements
// graph.Graph.
type SuiteBase struct {
	g graph.Graph
	suite.Suite
}

func (s *SuiteBase) SetGraph(g graph.Graph) {
	s.g = g
}

// TestUpsertLink verifies the link upsert logic.
func (s *SuiteBase) TestUpsertLink() {
	// Create a new link
	original := &graph.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Add(-10 * time.Hour),
	}

	s.Run("Upserting new link", func() {
		err := s.g.UpsertLink(original)
		s.Require().NoError(err, "Upserting new link should not return an error")
		s.Assert().NotNil(original.ID, "expected a linkID to be assigned to the new link")
	})

	s.Run("Update existing link with a newer timestamp and different URL", func() {
		accessedAt := time.Now().Truncate(time.Second).UTC()
		existing := &graph.Link{
			ID:          original.ID,
			URL:         "https://example.com",
			RetrievedAt: accessedAt,
		}

		err := s.g.UpsertLink(existing)
		s.Require().NoError(err, "Upserting existing link should not return an error")
		s.Assert().Equal(original.ID, existing.ID, "link ID changed while upserting")

		// Get the last upserted link
		stored, err := s.g.FindLink(existing.ID)
		s.Require().NoError(err, "Finding an existing link should not return an error")

		// Check the retrieved time if updated
		s.Assert().Equal(accessedAt, stored.RetrievedAt, "last accessed timestamp was not updated")

		// Attempt to insert a new link whose URL matches an existing link with
		// and provide an older accessedAt value
		sameURL := &graph.Link{
			URL:         existing.URL,
			RetrievedAt: time.Now().Add(-10 * time.Hour).UTC(), // This is older then the accessedAt
		}

		err = s.g.UpsertLink(sameURL)
		s.Require().NoError(err)
		s.Assert().Equal(existing.ID, sameURL.ID)

		// The timestamp should not overwritten
		stored, err = s.g.FindLink(existing.ID)
		s.Require().NoError(err)
		s.Assert().Equal(accessedAt, stored.RetrievedAt, "last accessed timestamp was overwritten with an older value")
	})
}

// TestUpsertEdge verifies the edge upsert logic.
func (s *SuiteBase) TestUpsertEdge() {
	// Create a links and Upsert it
	linkUUIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		s.Assert().NoError(s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
	}

	// Create an Edge
	edge := &graph.Edge{
		Src: linkUUIDs[0],
		Dst: linkUUIDs[1],
	}

	s.Run("Insert new edge", func() {
		// Upsert the new edge
		err := s.g.UpsertEdge(edge)
		s.Require().NoError(err)
		s.Assert().NotEqual(uuid.Nil, edge.ID)
		s.Assert().False(edge.UpdatedAt.IsZero())
	})

	s.Run("Update existing edge", func() {
		// Update existing edge
		other := &graph.Edge{
			ID: edge.ID,
			Src: linkUUIDs[0],
			Dst: linkUUIDs[1],
		}
		err := s.g.UpsertEdge(other)
		// Expactations:
		// 1. err is nil
		s.Require().NoError(err)
		// 2. the id for other and edge variables should be the same
		s.Assert().Equal(edge.ID, other.ID, "edge id changed while upserting")
		// 3. the update timestamp for other and edge variables should no
		//    be the same
		s.Assert().NotEqual(edge.UpdatedAt, other.UpdatedAt, "update at is not modified")
	})

	s.Run("Create edge with unknown link IDs", func() {
		bogus := &graph.Edge{
			Src: linkUUIDs[0],
			Dst: uuid.New(),
		}

		err := s.g.UpsertEdge(bogus)
		if s.Assert().Error(err) {
			s.Assert().True(xerrors.Is(err, graph.ErrUnknownEdgeLinks))
		}
	})
}
