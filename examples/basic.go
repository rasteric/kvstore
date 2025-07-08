package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/rasteric/kvstore"
)

func main() {
	db := kvstore.New()
	path, err := os.MkdirTemp("", "kvstore-test")
	if err != nil {
		panic(err)
	}
	err = db.Open(path)
	if err != nil {
		panic(err)
	}
	// make sure the test dir is cleaned up afterwards
	defer func() {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
		os.RemoveAll(path)
	}()

	// checking a key value pair doesn't exist
	if _, err := db.Get("hello"); errors.Is(err, kvstore.NotFoundErr) {
		fmt.Println(`there is no key "hello"`)
	} else {
		fmt.Println(err)
	}

	// setting a key value pair
	err = db.Set("example", "hello world!")
	if err != nil {
		panic(err)
	}

	// getting the value for a key
	s, err := db.Get("example")
	if err != nil {
		panic(err)
	}
	fmt.Println(s)

	// setting a default and key info
	err = db.SetDefault("example", "have a nice day!", kvstore.KeyInfo{Description: "an example key",
		Category: "testing"})
	if err != nil {
		panic(err)
	}

	// reverting a key value to its default
	if err := db.Revert("example"); err != nil {
		panic(err)
	}

	s, _ = db.Get("example")
	fmt.Println(s)
}
