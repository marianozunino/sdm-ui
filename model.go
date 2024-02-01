package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

type DataSource struct {
	Name    string
	Status  string
	Address string
	Type    string
	Tags    string
}

func (ds DataSource) Encode() []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(ds)
	if err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func (ds *DataSource) Decode(data []byte) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(ds)
	if err != nil {
		log.Fatal(err)
	}
}

func (ds DataSource) Key() []byte {
	return []byte(ds.Name)
}
