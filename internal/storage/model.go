package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type DataSource struct {
	Name    string
	Status  string
	Address string
	Type    string
	Tags    string
	WebURL  string
}

// Encode serializes the DataSource into a byte slice.
func (ds DataSource) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(ds); err != nil {
		return nil, fmt.Errorf("failed to encode DataSource: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode deserializes the byte slice into a DataSource.
func (ds *DataSource) Decode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(ds); err != nil {
		return fmt.Errorf("failed to decode DataSource: %w", err)
	}
	return nil
}

// Key returns the key for the DataSource, which is based on its Name.
func (ds DataSource) Key() []byte {
	return []byte(ds.Name)
}
