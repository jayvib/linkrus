package crawler

import (
	"context"
	"linkrus/pipeline"
	"linkrus/textindexer/index"
	"time"
)

var _ pipeline.Processor = (*textIndexer)(nil)

type textIndexer struct {
	indexer Indexer
}

func newTextIndexer(indexer Indexer) *textIndexer {
	return &textIndexer{
		indexer: indexer,
	}
}

func (t *textIndexer) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	doc := &index.Document{
		LinkID:    payload.LinkID,
		URL:       payload.URL,
		Title:     payload.Title,
		Content:   payload.TextContent,
		IndexedAt: time.Now(),
	}

	if err := t.indexer.Index(doc); err != nil {
		return nil, err
	}

	return p, nil
}
