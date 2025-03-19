package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rasteric/kvstore"
)

func main() {
	db := kvstore.New()
	dname, err := os.MkdirTemp("", "sampledir")
	if err != nil {
		panic(err)
	}
	err = db.Open(dname, "testdb")
	if err != nil {
		panic(err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
		os.RemoveAll(dname)
	}()

	err = db.Set("example", "hello world!")
	if err != nil {
		panic(err)
	}
	s, err := db.Get("example")
	if err != nil {
		panic(err)
	}
	fmt.Println(s)

	err = db.SetDefault("example", "have a nice day!", kvstore.KeyInfo{Description: "an example key",
		Category: "testing"})
	if err != nil {
		panic(err)
	}
	if err := db.Revert("example"); err != nil {
		panic(err)
	}
	s, _ = db.Get("example")
	fmt.Println(s)
}
