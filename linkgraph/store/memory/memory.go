package memory

import (
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"linkrus/linkgraph/graph"
	"sync"
	"time"
)

var _ graph.Graph = (*InMemoryGraph)(nil)

type edgeList []uuid.UUID

func NewInMemoryGraph() *InMemoryGraph {
	return &InMemoryGraph{
		links:        make(map[uuid.UUID]*graph.Link),
		edges:        make(map[uuid.UUID]*graph.Edge),
		linkURLIndex: make(map[string]*graph.Link),
		linkEdgeMap:  make(map[uuid.UUID]edgeList),
	}
}

type InMemoryGraph struct {
	mu sync.RWMutex

	links map[uuid.UUID]*graph.Link
	edges map[uuid.UUID]*graph.Edge

	linkURLIndex map[string]*graph.Link
	linkEdgeMap  map[uuid.UUID]edgeList
}

func (i *InMemoryGraph) UpsertLink(link *graph.Link) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if a link with the same URL already exists. If so,
	// convert this into an update and point the link ID to the
	// existing link.
	if existing := i.linkURLIndex[link.URL]; existing != nil {
		link.ID = existing.ID
		origTs := existing.RetrievedAt
		*existing = *link
		// Check if the new link timestamp is recent
		// than the existing timestamp
		if origTs.After(existing.RetrievedAt) {
			// So retain the existing timestamp
			existing.RetrievedAt = origTs
		}
		return nil
	}

	// Assign new ID and insert link
	for {
		link.ID = uuid.New()
		// Check if the uuid is already
		// exists
		if i.links[link.ID] == nil {
			break
		}
	}

	// Copy the original link
	// to avoid mutating it outside
	// this scope
	lCopy := new(graph.Link)
	*lCopy = *link

	// Set to the map
	i.linkURLIndex[lCopy.URL] = lCopy
	i.links[lCopy.ID] = lCopy

	return nil
}

func (i *InMemoryGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	link := i.links[id]
	if link == nil {
		return nil, xerrors.Errorf("find link: %w", graph.ErrNotFound)
	}

	// Copy the link
	lCopy := new(graph.Link)
	*lCopy = *link
	return lCopy, nil
}

func (i *InMemoryGraph) Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error) {
	panic("implement me")
}

func (i *InMemoryGraph) UpsertEdge(edge *graph.Edge) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if the source and destination ID
	// was already inserted
	_, srcExists := i.links[edge.Src]
	_, dstExists := i.links[edge.Dst]
	if !srcExists || !dstExists {
		return xerrors.Errorf("upsert edge: %w", graph.ErrUnknownEdgeLinks)
	}

	// Scan edge list from the source
	// Update its update timestamp to current time
	for _, edgeID := range i.linkEdgeMap[edge.Src] {
		// Get the edge object
		existingEdge := i.edges[edgeID]
		if existingEdge.Src == edge.Src && existingEdge.Dst == edge.Dst {
			existingEdge.UpdatedAt = time.Now()
			*edge = *existingEdge // to copy the existing details for an edge esp. ID
			return nil
		}
	}

	// Insert new edge
	// Find a unique UUID
	for {
		edge.ID = uuid.New()
		if i.edges[edge.ID] == nil {
			break
		}
	}

	edge.UpdatedAt = time.Now()

	// Copy the edge so that outside the
	// scope of this function won't
	// mutate the stored copy.
	eCopy := new(graph.Edge)
	*eCopy = *edge
	i.edges[eCopy.ID] = eCopy

	// Append the edge ID to the list of
	// edges originating from the edge's source
	// link.
	i.linkEdgeMap[edge.Src]	= append(i.linkEdgeMap[edge.Src], eCopy.ID)
	return nil
}

func (i *InMemoryGraph) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error) {
	panic("implement me")
}

func (i *InMemoryGraph) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	panic("implement me")
}
