package crawler

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"io"
	"linkrus/pipeline"
	"sync"
	"time"
)

var (
	_ pipeline.Payload = (*crawlerPayload)(nil)

	// Pool type attempts to reliev the pressure on garbage collector
	// by amortizing the cost of allocating objects across multiple clients.
	// When a client requests a new object from the pool, they can either receive a
	// cached instance or a newly allocated instance if the pool is empty.
	payloadPool = sync.Pool{
		New: func() interface{} {
			return new(crawlerPayload)
		},
	}
)

type crawlerPayload struct {
	// Will be filled with the input source
	LinkID      uuid.UUID
	URL         string
	RetrievedAt time.Time

	// The rest will be filled by the
	// pipeline stage

	// Populated by the link fetcher
	RawContent bytes.Buffer

	// Populated by the link extractor
	NoFollowLinks []string
	Links         []string

	// Populated by the text extractor
	Title       string
	TextContent string
}

func (c *crawlerPayload) Clone() pipeline.Payload {
	newP := payloadPool.Get().(*crawlerPayload)
	newP.LinkID = c.LinkID
	newP.URL = c.URL
	newP.RetrievedAt = c.RetrievedAt
	newP.NoFollowLinks = append([]string{}, c.NoFollowLinks[:]...)
	newP.Links = append([]string{}, c.Links[:]...)
	newP.Title = c.Title
	newP.TextContent = c.TextContent

	// Copy the body
	_, err := io.Copy(&newP.RawContent, &c.RawContent)
	if err != nil {
		panic(fmt.Sprintf("[Bug] error cloning payload raw content: %v", err))
	}
	return newP
}

func (c *crawlerPayload) MarkAsProcessed() {
	// Reset the fields to be reused
	c.LinkID = uuid.Nil
	c.RetrievedAt = time.Time{}
	c.URL = c.Title[:0] // <- optimization trick
	c.RawContent.Reset()
	c.NoFollowLinks = c.NoFollowLinks[:0]
	c.Links = c.Links[:0]
	c.Title = c.Title[:0]
	c.TextContent = c.TextContent[:0]
	payloadPool.Put(c)
}
