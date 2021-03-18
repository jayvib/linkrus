package main

import (
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	pb "linkrus/hacks/protobuf.tutorial/proto"
	"log"
)

func main() {
	p := &pb.Person{
		Id:    1,
		Name:  "Luffy Monkey",
		Email: "lmonkey@gmail.com",
		Phones: []*pb.Person_PhoneNumber{
			{Number: "555-1234", Type: pb.Person_HOME},
		},
	}

	out, err := proto.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("person.pb", out, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
