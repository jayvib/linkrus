package main

import (
	"context"
	"google.golang.org/grpc"
	pb "linkrus/hacks/grpc.tutorial/helloworld/proto"
	"log"
	"os"
	"time"
)

const (
	address     = "localhost:50001"
	defaultName = "world"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewGreeterClient(conn)

	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Greeting: %s", r.GetMessage())
}
