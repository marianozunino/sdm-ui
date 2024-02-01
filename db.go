package main

import (
	"os"
	"path"

	bolt "go.etcd.io/bbolt"
)

var DatasourceBucketKey []byte = []byte("datasource")

type storage struct {
	*bolt.DB
}

func newStorage() *storage {
	execPath, _ := os.Executable()
	execDir := path.Dir(execPath)
	db, err := bolt.Open(path.Join(execDir, "sources.db"), 0600, nil)
	if err != nil {
		panic(err)
	}
	return &storage{
		db,
	}
}

func (s *storage) Close() error {
	return s.DB.Close()
}

func (s *storage) storeServers(datasources []DataSource) error {
	err := s.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(DatasourceBucketKey)

		if err != nil {
			return err
		}

		for _, ds := range datasources {
			if err := b.Put(ds.Key(), ds.Encode()); err != nil {
				return err
			}
		}
		return err
	})

	return err
}

func (s *storage) retrieveDatasources() ([]DataSource, error) {
	datasources := []DataSource{}
	err := s.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(DatasourceBucketKey)
		if err != nil {
			return err
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			ds := DataSource{}
			ds.Decode(v)
			datasources = append(datasources, ds)
		}
		return nil
	})
	return datasources, err
}
