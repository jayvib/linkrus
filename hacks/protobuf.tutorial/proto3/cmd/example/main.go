package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"linkrus/hacks/protobuf.tutorial/proto3/product"
	"linkrus/hacks/protobuf.tutorial/proto3/supplier"
	"log"
)

func main() {

	sup := &supplier.Supplier{
		Id:   1,
		Name: "HP",
	}

	supAny, err := ptypes.MarshalAny(sup)
	if err != nil {
		log.Fatal(err)
	}

	prod := &product.Product{
		Id:       1,
		Name:     "Test Product",
		Status:   product.Product_ACTIVE,
		Supplier: supAny,
		Type:     &product.Product_IsMain{IsMain: true},
		Descriptions: map[int32]string{
			1: "The Short Description",
			2: "The Description 1",
			3: "The Description 2",
		},
	}

	buff, err := proto.Marshal(prod)
	if err != nil {
		log.Fatal(err)
	}

	prodCpy := new(product.Product)
	err = proto.Unmarshal(buff, prodCpy)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(prodCpy)
	fmt.Println(prodCpy.Type, prodCpy.GetIsMain())
}
