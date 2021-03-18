package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	pb "linkrus/hacks/grpc.tutorial/helloworld/proto"
	"log"
	"net"
)

const port = ":50001"

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, new(server))
	fmt.Println("Listening to port:", port)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
