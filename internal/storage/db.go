package storage

import (
	"fmt"
	"os"
	"path"

	bolt "go.etcd.io/bbolt"
)

var datasourceBucketKey []byte = []byte("datasource")

type Storage struct {
	*bolt.DB
}

func NewStorage() *Storage {
	execPath, _ := os.Executable()
	execDir := path.Dir(execPath)
	db, err := bolt.Open(path.Join(execDir, "sources.db"), 0600, nil)
	if err != nil {
		panic(err)
	}
	return &Storage{
		db,
	}
}

func buildBucketKey(email string) []byte {
	return []byte(fmt.Sprintf("%s:%s", email, datasourceBucketKey))
}

func (s *Storage) StoreServers(account string, datasources []DataSource) error {
	err := s.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(buildBucketKey(account))

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

func (s *Storage) RetrieveDatasources(account string) ([]DataSource, error) {
	datasources := []DataSource{}
	err := s.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(buildBucketKey(account))
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

func (s *Storage) Close() error {
	return s.DB.Close()
}
