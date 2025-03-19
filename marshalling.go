package kvstore

import (
	"bytes"
	"encoding/gob"
)

// MarshalBinary uses CBOR to marshal any CBOR-serializable value to a byte slice.
func MarshalBinary(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary assumes that a CBOR byte slice is given and nmarshals it into a variable
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
