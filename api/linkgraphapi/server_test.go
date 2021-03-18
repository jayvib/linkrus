package linkgraphapi_test

import (
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	gc "gopkg.in/check.v1"
	"linkrus/api/linkgraphapi/proto"
	"linkrus/linkgraph/graph"
)

var _ = gc.Suite(new(ServerTestSuite))
var minUUID = uuid.Nil
var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type ServerTestSuite struct {
	g graph.Graph

	netListener *bufconn.Listener
	grpcSrv     *grpc.Server

	cliConn *grpc.ClientConn
	cli     proto.LinkGraphClient
}

func (s *ServerTestSuite) SetUpTest(c *gc.C) {
	//s.g = memory.NewInMemoryGraph()
	//
	//s.netListener = bufconn.Listen(1024)
	//
	//s.grpcSrv = grpc.NewServer()
	//
	//proto.RegisterLinkGraphServer(s.grpcSrv, linkgraphapi.NewLinkGraphServer(s.g))
	//
	//go func() {
	//	err := s.grpcSrv.Serve(s.netListener)
	//	c.Assert(err, gc.NotNil)
	//}()
	//
	//var err error
	//s.cliConn, err = grpc.Dial(
	//	"bufnet",
	//	grpc.WithContextDialer(func(context.Context, string)(net.Conn, error) { return s.netListener.Dial() }),
	//	grpc.WithInsecure(),
	//)
	//
	//c.Assert(err, gc.IsNil)
	//
	//s.cli = proto.NewLinkGraphClient(s.cliConn)
}
