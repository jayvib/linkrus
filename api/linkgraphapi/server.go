package linkgraphapi

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"linkrus/api/linkgraphapi/proto"
	"linkrus/linkgraph/graph"
	"time"
)

var _ proto.LinkGraphServer = (*LinkGraphServer)(nil)

type LinkGraphServer struct {
	g graph.Graph
	proto.UnimplementedLinkGraphServer
}

func NewLinkGraphServer(g graph.Graph) *LinkGraphServer {
	return &LinkGraphServer{g: g}
}

func (l *LinkGraphServer) UpsertLink(_ context.Context, req *proto.Link) (*proto.Link, error) {
	var (
		err error

		// Initialize a link
		link = graph.Link{
			// with id
			ID: uuidFromBytes(req.Uuid),
			// with url
			URL: req.Url,
		}
	)

	// Convert from ptypes.Timestamp to time
	link.RetrievedAt, err = ptypes.Timestamp(req.RetrievedAt)
	if err != nil {
		return nil, err
	}

	// Upsert the link
	if err = l.g.UpsertLink(&link); err != nil {
		return nil, err
	}

	return req, nil
}

func (l *LinkGraphServer) UpsertEdge(_ context.Context, req *proto.Edge) (*proto.Edge, error) {
	edge := graph.Edge{
		ID:  uuidFromBytes(req.Uuid),
		Src: uuidFromBytes(req.SrcUuid),
		Dst: uuidFromBytes(req.DstUuid),
	}

	if err := l.g.UpsertEdge(&edge); err != nil {
		return nil, err
	}

	req.Uuid = edge.ID[:]
	req.SrcUuid = edge.Src[:]
	req.DstUuid = edge.Dst[:]
	req.UpdatedAt = timeToProto(edge.UpdatedAt)

	return req, nil
}

func (l *LinkGraphServer) RemoveStaleEdges(ctx context.Context, query *proto.RemoveStaleEdgesQuery) (*empty.Empty, error) {
	panic("implement me")
}

func (l *LinkGraphServer) Links(r *proto.Range, server proto.LinkGraph_LinksServer) error {
	panic("implement me")
}

func (l *LinkGraphServer) Edges(r *proto.Range, server proto.LinkGraph_EdgesServer) error {
	panic("implement me")
}

func uuidFromBytes(b []byte) uuid.UUID {
	if len(b) != 16 {
		return uuid.Nil
	}
	var dst uuid.UUID
	copy(dst[:], b)
	return dst
}

func timeToProto(t time.Time) *timestamp.Timestamp {
	ts, _ := ptypes.TimestampProto(t)
	return ts
}
