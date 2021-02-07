package crawler

import (
	"context"
	"github.com/sirupsen/logrus"
	"linkrus/linkgraph/graph"
	"linkrus/pipeline"
	"time"
)

func newGraphUpdater(g Graph) *graphUpdater {
	return &graphUpdater{
		updater: g,
	}
}

var _ pipeline.Processor = (*graphUpdater)(nil)

type graphUpdater struct {
	updater Graph
}

func (g *graphUpdater) Process(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	// Create a new link
	src := &graph.Link{
		ID:          payload.LinkID,
		URL:         payload.URL,
		RetrievedAt: time.Now(),
	}

	logrus.Trace("Upserting source link")
	// Upsert the source link
	if err := g.updater.UpsertLink(src); err != nil {
		return nil, err
	}

	// Upsert discovered no-follow links without creating an edge
	for _, dstLink := range payload.NoFollowLinks {
		dst := &graph.Link{URL: dstLink}
		if err := g.updater.UpsertLink(dst); err != nil {
			return nil, err
		}
	}

	// Upsert discovered links and create edges for them. Keep
	// track of the current time so we can drop stale edges that
	// have not been updated after this loop.
	removeEdgesOlderThan := time.Now()
	logrus.Debug(payload.Links)
	for _, dstLink := range payload.Links {

		// Upsert the discovered link
		dst := &graph.Link{URL: dstLink}
		logrus.Trace("Upserting:", dstLink)
		if err := g.updater.UpsertLink(dst); err != nil {
			return nil, err
		}

		logrus.Trace(dst.ID, dst.URL)
		// Upsert the discovered link as edge
		if err := g.updater.UpsertEdge(&graph.Edge{Src: src.ID, Dst: dst.ID}); err != nil {
			return nil, err
		}
	}

	// Drop stale edges that were not touched while upserting the
	// outgoing edges.
	if err := g.updater.RemoveStaleEdges(src.ID, removeEdgesOlderThan); err != nil {
		return nil, err
	}

	return p, nil
}
