package storage

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

var datasourceBucketKey = []byte("datasource")

type Storage struct {
	*bolt.DB
	account string
}

// NewStorage initializes and returns a new Storage instance, ensuring that the bucket for the given account is created.
func NewStorage(account string, path string) (*Storage, error) {
	dbPath := filepath.Join(path, "sdm-sources.db")
	log.Debug().Msgf("Opening database at %s", dbPath)

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{
		DB:      db,
		account: account,
	}

	if err := storage.ensureBucketExists(); err != nil {
		return nil, err
	}

	return storage, nil
}

// ensureBucketExists ensures that the bucket for the given account exists in the database.
func (s *Storage) ensureBucketExists() error {
	bucketKey := buildBucketKey(s.account)
	return s.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketKey)
		if err != nil {
			return fmt.Errorf("failed to create bucket for account %s with key %s: %w", s.account, bucketKey, err)
		}
		return nil
	})
}

// buildBucketKey constructs a bucket key using the account and the datasource bucket key.
func buildBucketKey(account string) []byte {
	return []byte(fmt.Sprintf("%s:%s", account, datasourceBucketKey))
}

// StoreServers stores the provided datasources for the specified account.
func (s *Storage) StoreServers(datasources []DataSource) error {
	return s.Update(func(tx *bolt.Tx) error {
		log.Debug().Msgf("Storing %d datasources", len(datasources))

		bucket := tx.Bucket(buildBucketKey(s.account))
		if bucket == nil {
			return fmt.Errorf("bucket for account %s not found", s.account)
		}

		for _, ds := range datasources {
			// Encode the DataSource and handle any errors
			encodedData, err := ds.Encode()
			if err != nil {
				return fmt.Errorf("failed to encode datasource %s: %w", ds.Name, err)
			}

			// Store the encoded DataSource in the bucket
			if err := bucket.Put(ds.Key(), encodedData); err != nil {
				return fmt.Errorf("failed to store datasource %s: %w", ds.Name, err)
			}
		}

		log.Debug().Msgf("Successfully stored %d datasources", len(datasources))
		return nil
	})
}

// RetrieveDatasources retrieves all datasources for the specified account.
func (s *Storage) RetrieveDatasources() ([]DataSource, error) {
	var datasources []DataSource
	bucketKey := buildBucketKey(s.account)

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return fmt.Errorf("bucket for account %s not found", s.account)
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var ds DataSource
			if err := ds.Decode(v); err != nil {
				log.Error().Msgf("Failed to decode datasource: %v", err)
				continue
			}
			datasources = append(datasources, ds)
		}

		return nil
	})

	return datasources, err
}

// GetDatasource retrieves a single datasource by name for the specified account.
func (s *Storage) GetDatasource(name string) (DataSource, error) {
	var datasource DataSource
	bucketKey := buildBucketKey(s.account)

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return fmt.Errorf("bucket for account %s not found", bucketKey)
		}

		value := bucket.Get([]byte(name))
		if value == nil {
			return fmt.Errorf("datasource %s not found", name)
		}

		if err := datasource.Decode(value); err != nil {
			return fmt.Errorf("failed to decode datasource %s: %w", name, err)
		}

		return nil
	})

	return datasource, err
}
