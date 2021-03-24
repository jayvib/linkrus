package main

import (
	"context"
	"google.golang.org/grpc"
	pb "linkrus/hacks/grpc.tutorial/streaming/proto"
	"log"
	"sync"
)

// Tutorial:
// https://www.freecodecamp.org/news/grpc-server-side-streaming-with-go/

func main() {

	// Connect to the server
	conn, err := grpc.Dial(":50005", grpc.WithInsecure())
	mustNoErr(err)
	defer conn.Close()

	// Create a client using the generated protobuf files
	client := pb.NewStreamServiceClient(conn)
	in := &pb.Request{Id: 1}

	// Do a requeset
	stream, err := client.FetchResponse(context.Background(), in)
	mustNoErr(err)

	var wg sync.WaitGroup

	// Listen from the server
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}

			log.Printf("Resp received: %s", resp.Result)
		}
	}()

	wg.Wait()
	log.Println("Finished")
}

func mustNoErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
