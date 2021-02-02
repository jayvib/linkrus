package crawler

import (
	"context"
	"github.com/google/uuid"
	"linkrus/linkgraph/graph"
	"linkrus/pipeline"
	"linkrus/textindexer/index"
	"net/http"
	"time"
)

//go:generate mockgen -package mocks -destination mocks/mocks.go . URLGetter,PrivateNetworkDetector,Graph,Indexer

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by objects that can detect
// whether a host resolves to a private network address.
type PrivateNetworkDetector interface {
	IsPrivate(host string) (bool, error)
}

// Graph is implemented by objects that can upsert links and edges into a link
type Graph interface {
	// UpsertLInk creates a new link or updates an existing link
	UpsertLink(link *graph.Link) error

	// UpsertEdge creates a new edge or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error

	// RemoveStaleEdges removes any edge that originates from the
	// specified link ID and was updated before the specified timestamp.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

type Indexer interface {
	Index(doc *index.Document) error
}

type linkSource struct {
	linkIt graph.LinkIterator
}

func (l *linkSource) Next(_ context.Context) bool {
	return l.linkIt.Next()
}

func (l *linkSource) Payload() pipeline.Payload {
	// Get the link
	link := l.linkIt.Link()

	// Get a crawlerPayload from the pool
	p := payloadPool.Get().(*crawlerPayload)

	// Populate the crawlerPayload.
	p.LinkID = link.ID
	p.URL = link.URL
	p.RetrievedAt = link.RetrievedAt
	return p
}

func (l *linkSource) Error() error {
	return l.linkIt.Error()
}

type nopSink struct{}

func (nopSink) Consume(context.Context, pipeline.Payload) error {
	return nil
}
