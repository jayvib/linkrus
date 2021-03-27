package linkgraphapi_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	gc "gopkg.in/check.v1"
	"io"
	"linkrus/api/linkgraphapi"
	"linkrus/api/linkgraphapi/proto"
	"linkrus/linkgraph/graph"
	"linkrus/linkgraph/store/memory"
	"net"
	"time"
)

var _ = gc.Suite(new(ServerTestSuite))
var minUUID = uuid.Nil
var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type ServerTestSuite struct {
	g graph.Graph

	// Network listener
	netListener *bufconn.Listener
	// Grpc server
	grpcSrv *grpc.Server

	// Grpc client connection
	cliConn *grpc.ClientConn

	// Client
	cli proto.LinkGraphClient
}

func (s *ServerTestSuite) SetUpTest(c *gc.C) {
	s.g = memory.NewInMemoryGraph()

	s.netListener = bufconn.Listen(1024)

	s.grpcSrv = grpc.NewServer()

	proto.RegisterLinkGraphServer(s.grpcSrv, linkgraphapi.NewLinkGraphServer(s.g))

	go func() {
		err := s.grpcSrv.Serve(s.netListener)
		c.Assert(err, gc.NotNil)
	}()

	var err error
	s.cliConn, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return s.netListener.Dial() }),
		grpc.WithInsecure(),
	)

	c.Assert(err, gc.IsNil)

	s.cli = proto.NewLinkGraphClient(s.cliConn)
}

func (s *ServerTestSuite) TearDownTest(_ *gc.C) {
	_ = s.cliConn.Close()
	s.grpcSrv.Stop()
	_ = s.netListener.Close()
}

func (s *ServerTestSuite) TestInsertLink(c *gc.C) {
	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Url:         "http://example.com",
		RetrievedAt: mustEncodeTimestamp(c, now),
	}
	res, err := s.cli.UpsertLink(context.TODO(), req)
	c.Assert(err, gc.IsNil)

	c.Assert(res.Uuid, gc.Not(gc.DeepEquals), req.Uuid, gc.Commentf("UUID not assigned to new link"))
	c.Assert(mustDecodeTimestamp(c, res.RetrievedAt), gc.Equals, now)
}

func (s *ServerTestSuite) TestUpdateLink(c *gc.C) {

	// Insert first a link
	link := &graph.Link{URL: "http://example.com"}
	c.Assert(s.g.UpsertLink(link), gc.IsNil)

	// Update the link
	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         "http://example.com",
		RetrievedAt: mustEncodeTimestamp(c, now),
	}

	// Do request to the GRPC server
	res, err := s.cli.UpsertLink(context.TODO(), req)
	c.Assert(err, gc.IsNil)

	// Assert the id
	c.Assert(req.Uuid, gc.DeepEquals, res.Uuid, gc.Commentf("UUID for existing link modified"))

	// Assert the URL
	c.Assert(req.Url, gc.Equals, res.Url, gc.Commentf("URL not updated"))

	// Assert the timestamp
	c.Assert(mustDecodeTimestamp(c, res.RetrievedAt), gc.Equals, now)
}

func (s *ServerTestSuite) TestInsertEdge(c *gc.C) {
	// Add two links to the graph
	src := &graph.Link{URL: "http://example.com"}
	dst := &graph.Link{URL: "http://foo.com"}

	// In order to set the ID of above links
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst), gc.IsNil)

	// Create an Edge
	req := &proto.Edge{
		SrcUuid: src.ID[:],
		DstUuid: dst.ID[:],
	}

	res, err := s.cli.UpsertEdge(context.TODO(), req)

	c.Assert(err, gc.IsNil)
	c.Assert(res.Uuid[:], gc.Not(gc.DeepEquals), req.Uuid[:], gc.Commentf("UUID not assigned to new edge"))
	c.Assert(res.DstUuid[:], gc.DeepEquals, req.DstUuid[:])
	c.Assert(res.SrcUuid[:], gc.DeepEquals, req.SrcUuid[:])
	c.Assert(res.UpdatedAt, gc.Not(gc.Equals), 0)
}

func (s *ServerTestSuite) TestUpdateEdge(c *gc.C) {
	// Create a 2 new link for source the destination
	src := &graph.Link{URL: "http://example.com"}
	dst := &graph.Link{URL: "http://foo.com"}
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst), gc.IsNil)

	// Create an edge
	edge := &graph.Edge{
		Src: src.ID,
		Dst: dst.ID,
	}
	c.Assert(s.g.UpsertEdge(edge), gc.IsNil)

	// Update the edge
	req := &proto.Edge{
		Uuid:    edge.ID[:],
		SrcUuid: src.ID[:],
		DstUuid: dst.ID[:],
	}

	res, err := s.cli.UpsertEdge(context.TODO(), req)
	c.Assert(err, gc.IsNil)
	// Should be equal
	c.Assert(res.Uuid, gc.DeepEquals, req.Uuid, gc.Commentf("UUID for existing edge modified"))
	c.Assert(res.DstUuid[:], gc.DeepEquals, req.DstUuid[:])
	c.Assert(res.SrcUuid[:], gc.DeepEquals, req.SrcUuid[:])

	// Assert the date. And the res.UpdatedAt must be after the edge.UpdatedAt
	c.Assert(mustDecodeTimestamp(c, res.UpdatedAt).After(edge.UpdatedAt), gc.Equals, true)
}

func (s *ServerTestSuite) TestLinks(c *gc.C) {

	// Insert first a link to the graph
	sawLinks := make(map[uuid.UUID]bool)
	for i := 0; i < 100; i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), gc.IsNil)
		sawLinks[link.ID] = false
	}

	filter := mustEncodeTimestamp(c, time.Now().Add(time.Hour))
	stream, err := s.cli.Links(
		context.TODO(),
		&proto.Range{
			FromUuid: minUUID[:],
			ToUuid:   maxUUID[:],
			Filter:   filter,
		},
	)
	c.Assert(err, gc.IsNil)

	for {
		next, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		linkID, err := uuid.FromBytes(next.Uuid)
		c.Assert(err, gc.IsNil)

		alreadySeen, exists := sawLinks[linkID]
		if !exists {
			c.Fatalf("saw unexpected link with ID %q", linkID)
		} else if alreadySeen {
			c.Fatalf("saw duplicate link with ID %q", linkID)
		}
		sawLinks[linkID] = true
	}

	for linkID, seen := range sawLinks {
		if !seen {
			c.Fatalf("expected to see link with ID %q", linkID)
		}
	}
}

func (s *ServerTestSuite) TestEdges(c *gc.C) {
	// Add a link and edges to the graph
	links := make([]uuid.UUID, 100)
	for i := 0; i < len(links); i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), gc.IsNil)
		links[i] = link.ID
	}

	sawEdges := make(map[uuid.UUID]bool)
	for i := 0; i < len(links); i++ {
		edge := &graph.Edge{
			Src: links[0],
			Dst: links[i],
		}
		c.Assert(s.g.UpsertEdge(edge), gc.IsNil)
		sawEdges[edge.ID] = false
	}

	filter := mustEncodeTimestamp(c, time.Now().Add(time.Hour))

	stream, err := s.cli.Edges(context.TODO(), &proto.Range{
		FromUuid: minUUID[:],
		ToUuid:   maxUUID[:],
		Filter:   filter,
	})
	c.Assert(err, gc.IsNil)

	// Collect the stream
	for {
		next, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		edgeID, err := uuid.FromBytes(next.Uuid)
		c.Assert(err, gc.IsNil)

		alreadySeen, exists := sawEdges[edgeID]
		if !exists {
			c.Fatalf("saw unexpected edges with ID %q", edgeID)
		} else if alreadySeen {
			c.Fatalf("saw duplicate edge with ID %q", edgeID)
		}

		sawEdges[edgeID] = true
	}

	// Check if there's edges that haven't seen
	for edgeID, seen := range sawEdges {
		if !seen {
			c.Fatalf("expected to see edge with ID %q", edgeID)
		}
	}
}

func (s *ServerTestSuite) TestRetainVersionedEdges(c *gc.C) {
	// Add three links and two edges to the graph with different versions.
	src := &graph.Link{URL: "http://example.com"}
	dst1 := &graph.Link{URL: "http://foo.com"}
	dst2 := &graph.Link{URL: "http://bar.com"}
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst1), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst2), gc.IsNil)

	// Upsert a first edge
	edge1 := &graph.Edge{Src: src.ID, Dst: dst1.ID}
	c.Assert(s.g.UpsertEdge(edge1), gc.IsNil)

	// Simulate time-passed-by.
	time.Sleep(100 * time.Millisecond)
	t1 := time.Now() // Capture time now
	edge2 := &graph.Edge{Src: src.ID, Dst: dst2.ID}
	c.Assert(s.g.UpsertEdge(edge2), gc.IsNil)

	// Do a RemoveStaleEdges request
	req := &proto.RemoveStaleEdgesQuery{
		FromUuid:      src.ID[:],
		UpdatedBefore: mustEncodeTimestamp(c, t1),
	}

	_, err := s.cli.RemoveStaleEdges(context.TODO(), req)
	c.Assert(err, gc.IsNil)

	// Do assertions
	// Query all the edges upto current time
	iter, err := s.g.Edges(minUUID, maxUUID, time.Now())
	c.Assert(err, gc.IsNil)

	// Iterate the iterator and check if the edge1 ID is
	// not found.
	var edgeCount int
	for iter.Next() {
		edgeCount++
		edge := iter.Edge()
		c.Assert(edge.ID, gc.Not(gc.DeepEquals), edge1.ID, gc.Commentf("expected edge1 to be dropper"))
	}

	c.Assert(iter.Error(), gc.IsNil)
	c.Assert(edgeCount, gc.Equals, 1)
}
