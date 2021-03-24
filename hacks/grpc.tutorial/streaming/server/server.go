package main

import (
	"fmt"
	"google.golang.org/grpc"
	protobuf "linkrus/hacks/grpc.tutorial/streaming/proto"
	"log"
	"net"
	"sync"
	"time"
)

type server struct {
	protobuf.UnimplementedStreamServiceServer
}

func (server) FetchResponse(in *protobuf.Request, srv protobuf.StreamService_FetchResponseServer) error {

	log.Printf("fetch response for id: %d\n", in.Id)

	// Create a 5 goroutines to send the data
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(count int) {
			defer wg.Done()

			// Sleep to simulate work
			time.Sleep(time.Duration(count) * time.Second)

			resp := &protobuf.Response{
				Result: fmt.Sprintf("Request #%d for Id:%d", count, in.Id),
			}

			if err := srv.Send(resp); err != nil {
				log.Printf("got an error %v", err)
			}
			log.Printf("finishing request number: %d\n", count)
		}(i)
	}

	wg.Wait()
	return nil
}

func main() {
	lis, err := net.Listen("tcp", ":50005")
	if err != nil {
		log.Fatal(err)
	}

	defer lis.Close()

	s := grpc.NewServer()
	protobuf.RegisterStreamServiceServer(s, server{})

	log.Println("start server")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
