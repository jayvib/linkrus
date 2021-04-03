package linkgraphapi

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"linkrus/api/linkgraphapi/proto"
	"linkrus/linkgraph/graph"
	"time"
)

//go:generate mockgen -package mocks -destination mocks/mock.go ./proto LinkGraphClient,LinkGraph_LinksClient,LinkGraph_EdgesClient

func NewLinkGraphClient(ctx context.Context, rpcClient proto.LinkGraphClient) *LinkGraphClient {
	return &LinkGraphClient{
		ctx: ctx,
		cli: rpcClient,
	}
}

type LinkGraphClient struct {
	ctx context.Context

	// GRPC Client. This is the dependency that needs to be
	// mocked.
	cli proto.LinkGraphClient
}

func (c *LinkGraphClient) UpsertLink(link *graph.Link) error {

	// Convert to proto message
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         link.URL,
		RetrievedAt: timeToProto(link.RetrievedAt),
	}

	// Do a GRPC calls to the server
	res, err := c.cli.UpsertLink(c.ctx, req)
	if err != nil {
		return err
	}

	// Convert from proto message back to link.
	link.ID = uuidFromBytes(res.Uuid)
	link.URL = res.Url

	link.RetrievedAt = res.RetrievedAt.AsTime()

	return nil
}

func (c *LinkGraphClient) UpsertEdge(edge *graph.Edge) error {
	// Convert to proto message
	req := &proto.Edge{
		Uuid:    edge.ID[:],
		SrcUuid: edge.Src[:],
		DstUuid: edge.Dst[:],
	}

	res, err := c.cli.UpsertEdge(c.ctx, req)
	if err != nil {
		return err
	}

	edge.ID = uuidFromBytes(res.Uuid)
	edge.UpdatedAt = res.UpdatedAt.AsTime()
	return nil
}

func (c *LinkGraphClient) Links(fromUUID, toUUID uuid.UUID, before time.Time) (graph.LinkIterator, error) {
	filter := timestamppb.New(before)

	req := &proto.Range{
		FromUuid: fromUUID[:],
		ToUuid:   toUUID[:],
		Filter:   filter,
	}

	ctx, cancelFn := context.WithCancel(context.TODO())
	stream, err := c.cli.Links(ctx, req)
	if err != nil {
		cancelFn()
		return nil, err
	}

	return &linkIterator{stream: stream, cancelFn: cancelFn}, nil
}

func (c *LinkGraphClient) Edges(from, to uuid.UUID, before time.Time) (graph.EdgeIterator, error) {
	filter := timestamppb.New(before)

	ctx, cancelFn := context.WithCancel(c.ctx)
	stream, err := c.cli.Edges(ctx, &proto.Range{FromUuid: from[:], ToUuid: to[:], Filter: filter})
	if err != nil {
		cancelFn()
		return nil, err
	}

	return &edgeIterator{stream: stream, cancelFn: cancelFn}, nil
}

var _ graph.LinkIterator = (*linkIterator)(nil)

type linkIterator struct {
	stream  proto.LinkGraph_LinksClient
	next    *graph.Link
	lastErr error

	cancelFn func()
}

func (l *linkIterator) Next() bool {
	res, err := l.stream.Recv()
	if err != nil {
		if err != io.EOF {
			l.lastErr = err
		}
		l.cancelFn()
		return false
	}

	lastAccessed := res.RetrievedAt.AsTime()

	l.next = &graph.Link{
		ID:          uuidFromBytes(res.Uuid[:]),
		URL:         res.Url,
		RetrievedAt: lastAccessed,
	}

	return true
}

func (l *linkIterator) Error() error {
	return l.lastErr
}

func (l *linkIterator) Close() error {
	l.cancelFn()
	return nil
}

func (l *linkIterator) Link() *graph.Link {
	return l.next
}

type edgeIterator struct {
	stream  proto.LinkGraph_EdgesClient
	next    *graph.Edge
	lastErr error

	cancelFn func()
}

func (i *edgeIterator) Next() bool {
	res, err := i.stream.Recv()
	if err != nil {
		if err != io.EOF {
			i.lastErr = err
		}
		i.cancelFn()
		return false
	}

	lastAccessed := res.UpdatedAt.AsTime()

	i.next = &graph.Edge{
		ID:        uuidFromBytes(res.Uuid[:]),
		Src:       uuidFromBytes(res.SrcUuid[:]),
		Dst:       uuidFromBytes(res.DstUuid[:]),
		UpdatedAt: lastAccessed,
	}

	return true
}

func (i *edgeIterator) Error() error {
	return i.lastErr
}

func (i *edgeIterator) Close() error {
	i.cancelFn()
	return nil
}

func (i *edgeIterator) Edge() *graph.Edge {
	return i.next
}
