package kvstore

import (
	"bytes"
	"encoding/gob"
)

// MarshalBinary uses gob encoding to marshal a value to a byte slice. To encode
// structs, use gob.Register(yourstruct{}) to register them.
func MarshalBinary(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary assumes that a gob encoded byte slice is given and nmarshals it into a variable
// of the empty interface type, returns an error if the data is malformed.
func UnmarshalBinary(b []byte) (any, error) {
	dec := gob.NewDecoder(bytes.NewReader(b))
	var v any
	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return v, nil
}
