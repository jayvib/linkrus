package main

//
//import (
//	"fmt"
//	"github.com/golang/protobuf/proto"
//	"io/ioutil"
//	pb "linkrus/hacks/protobuf.tutorial/proto"
//	"log"
//)
//
//func main() {
//	file := "person.pb"
//
//	in, err := ioutil.ReadFile(file)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	person := new(pb.Person)
//	err = proto.Unmarshal(in, person)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(person)
//}
