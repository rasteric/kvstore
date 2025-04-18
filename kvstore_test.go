package kvstore

import (
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

type testStruct struct {
	A int32
	B int16
	C byte
	D string
	E []byte
}

func TestKVStore(t *testing.T) {
	db := New()
	path, err := os.MkdirTemp("", "kvstore-test")
	if err != nil {
		t.Errorf(`failed to create tempdir: %v`, err)
	}
	err = db.Open(path)
	if err != nil {
		t.Errorf(`failed to open database: %v`, err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Errorf(`failed to close database: %v`, err)
		}
		os.RemoveAll(path)
	}()
	var x int = 10
	err = db.Set("hello", x)
	if err != nil {
		t.Errorf(`failed to set integer: %v`, err)
	}
	n, err := db.Get("hello")
	if err != nil {
		t.Errorf(`failed to get integer: %v`, err)
	}
	if n.(int) != 10 {
		t.Errorf(`expected 10, got %v`, n)
	}
	var y uint32 = 11
	err = db.Set("hello", y)
	if err != nil {
		t.Errorf(`failed to set integer: %v`, err)
	}
	a, err := db.Get("hello")
	if err != nil {
		t.Errorf(`failed to get integer: %v`, err)
	}
	if a.(uint32) != 11 {
		t.Errorf(`expected 10, got %v`, a)
	}
	err = db.Set("hello", "whatever")
	if err != nil {
		t.Errorf(`failed to set string key: %v`, err)
	}
	s, err := db.Get("hello")
	if s.(string) != "whatever" {
		t.Errorf(`failed to get string key: %v`, err)
	}
	err = db.SetDefault("hello", "world", KeyInfo{Description: "a default key", Category: "tests"})
	if err != nil {
		t.Errorf(`failed to set default: %v`, err)
	}
	s, err = db.Get("hello")
	if s.(string) != "whatever" {
		t.Errorf(`failed to get string value after setting default: %v`, err)
	}
	info, ok := db.Info("hello")
	if !ok {
		t.Errorf(`failed to get key info when there should be one`)
	}
	if info.Description != "a default key" || info.Category != "tests" {
		t.Errorf(`wrong key info returned`)
	}
	err = db.Revert("hello")
	if err != nil {
		t.Errorf(`failed to revert a key to its default (though there should be a default): %v`, err)
	}
	s, err = db.Get("hello")
	if err != nil {
		t.Errorf(`failed to get string value: %v`, err)
	}
	if s.(string) != "world" {
		t.Errorf(`failed to revert to default`)
	}
	gob.Register(testStruct{})
	b := testStruct{
		A: 10,
		B: 20,
		C: 30,
		D: "hello",
		E: []byte("world"),
	}
	err = db.Set("test", b)
	if err != nil {
		t.Errorf(`failed to set test struct: %v`, err)
	}
	c, err := db.Get("test")
	if err != nil {
		t.Errorf(`failed to get test struct: %v`, err)
	}
	if _, ok := c.(testStruct); !ok {
		t.Errorf(`failed to get test struct: %v`, err)
	}
	e := c.(testStruct)
	if e.A != 10 || e.B != 20 || e.C != 30 || e.D != "hello" || string(e.E) != "world" {
		t.Errorf(`wrong data after getting testStruct`)
	}
	// stress test a bit
	for i := 0; i < 1000; i++ {
		key, _ := generateRandomHex(16)
		value, _ := generateRandomHex(16)
		err := db.Set(key, value)
		if err != nil {
			t.Errorf(`failed to set random key value pair: %v`, err)
		}
		retrieved, err := db.Get(key)
		if err != nil {
			t.Errorf(`failed to retrieve a random value: %v`, err)
		}
		if retrieved.(string) != value {
			t.Errorf(`retrieved wrong value for random key value pair`)
		}
		db.Delete(key)
		if _, err := db.Get(key); err == nil {
			t.Errorf(`value was returned for deleted key value pair`)
		}
	}
	// test map methods
	testPairs := make(map[string]any)
	for i := 0; i < 1000; i++ {
		key, _ := generateRandomHex(16)
		value, _ := generateRandomHex(16)
		testPairs[key] = value
	}
	err = db.SetMany(testPairs)
	if err != nil {
		t.Errorf(`failed to set many: %v`, err)
	}
	part, err := db.GetAll(100)
	if err != nil {
		t.Errorf(`getting all data with limit failed: %v`, err)
	}
	if len(part) != 100 {
		t.Errorf(`limit was set to 100 but got %v key value pairs`, len(part))
	}
	everyThing, err := db.GetAll(0)
	for k, v := range testPairs {
		if everyThing[k] != v {
			t.Errorf(` get all result does not match get many, got %v expected %v`,
				everyThing[k], v)
			break
		}
	}
	keys := make([]string, 0, len(everyThing))
	for k, _ := range everyThing {
		keys = append(keys, k)
	}
	err = db.DeleteMany(keys)
	if err != nil {
		t.Errorf(`error with delete many: %v`, err)
	}
	for k, _ := range testPairs {
		if _, err := db.Get(k); !errors.Is(err, NotFoundErr) {
			t.Errorf(`expected key to be deleted but it wasn't: %v`, k)
		}
	}
}

// generateRandomHex uses crypt.Rand to generate n random bytes
// and returns them as string.
func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
