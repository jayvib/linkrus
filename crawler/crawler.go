package crawler

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

// Config encapsulates the configuration options for creating a new Crawler.
type Config struct {
	// A PrivateNetworkDetector instance
	PrivateNetworkDetector PrivateNetworkDetector

	// A URLGetter instance for fetching links.
	URLGetter URLGetter

	// A GraphUpdater instance for adding ne links to the link graph.
	Graph Graph

	// A TextIndexer instance for indexing the content of each retrieved link.
	Indexer Indexer

	// The number of concurrent workers used for retrieving links.
	FetchWorkers int
}

// Crawler implements a web-page crawling pipeline consisting of the
// following stages:
//
// - Given a URL, retrieve the web-page contents from the remote server.
// - Extract and resolve absolute and relative links from the retrieved page.
// - Extract page title and text content from the retrieved page.
// - Update the link graph: add new links and create edges between the crawled
//   page and links within it.
// - Index crawled page title and text content.
type Crawler struct {
	p *pipeline.Pipeline
}

// NewCrawler returns a new crawler instance
func NewCrawler(cfg Config) *Crawler {
	return &Crawler{
		p: assembleCrawlerPipeline(cfg),
	}
}

func assembleCrawlerPipeline(cfg Config) *pipeline.Pipeline {
	return pipeline.New(
		pipeline.FixedWorkerPool(newLinkFetcher(cfg.URLGetter, cfg.PrivateNetworkDetector), cfg.FetchWorkers),
		pipeline.FIFO(newLinkExtractor(cfg.PrivateNetworkDetector)),
		pipeline.FIFO(newTextExtractor()),
		pipeline.Broadcast(
			newGraphUpdater(cfg.Graph),
			newTextIndexer(cfg.Indexer),
		),
	)
}

func (c *Crawler) Crawl(ctx context.Context, linkIt graph.LinkIterator) (int, error) {
	sink := new(countingSink)
	err := c.p.Process(ctx, &linkSource{linkIt: linkIt}, sink)
	return sink.getCount(), err
}

type countingSink struct {
	count int
}

func (s *countingSink) Consume(_ context.Context, p pipeline.Payload) error {
	payload := p.(*crawlerPayload)
	logrus.Debug(payload.Links)
	s.count++
	return nil
}

func (s *countingSink) getCount() int {
	return s.count / 2
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
