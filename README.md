![KVStore](logo.png)

[![GoDoc](https://godoc.org/github.com/rasteric/kvstore/go?status.svg)](https://godoc.org/github.com/rasteric/kvstore)
[![Go Report Card](https://goreportcard.com/badge/github.com/rasteric/kvstore)](https://goreportcard.com/report/github.com/rasteric/kvstore)

__KVStore is an Sqlite3-backed embedded local key value store for Go, focusing on simplicity and data integrity. It currently hardcodes the CGO-free Sqlite3 library `github.com/ncruces/go-sqlite3` in WAL mode as backend.__

## Installation

`go get https://github.com/rasteric/kvstore`

## Usage 

```
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

	err = db.SetDefault("example", "have a nice day!",
        kvstore.KeyInfo{Description: "an example key",
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
```

Notice that `kvstore.NotFoundErr` is returned when a `get` operation fails. Since all kinds of errors can occur with file-based databases, this API was chosen instead of the more common `value, ok:=db.Get(key)` from maps and other key value stores. Check for the error with `errors.Is(err,kvstore.NotFoundErr)` to distinguish it from other errors. Use `SetDefault` to set a default, in case of which the default is returned if no value was set.

## Documentation

All API calls are in the following interface:

```
// KeyValueStore is the interface for a key value database.
type KeyValueStore interface {
	Open(path, name string) error     // open the database in path/name/
	Close() error                     // close the database
	Set(key string, value any) error  // set a key to a value
	SetDefault(key string, value any, // set a default and info for a key 
		info KeyInfo) error           
	Get(key string) (any, error)     // get the value for a key
	Revert(key string) error         // revert a value to its default
	Info(key string) (KeyInfo, bool) // return key information for a key if present
	Delete(key string)               // remove a key value pair
}
```

## Endoding

This library uses Go's gob encoding to encode values in the database. This means that you have to use `gob.Register(mystruct{})` if you want to store values of custom struct `mystruct` in the key value database. It also means that all limitations of gob encoding apply to the values stored.

## License

This library is MIT licensed and free for commercial and personal use as long as the license conditions are satisfied. See the accompanying LICENSE file for more information.
