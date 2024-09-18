package storage

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

const (
	datasourceBucketPrefix = "datasource"
	currentDBVersion       = 2 // increment this whenever the database schema changes
	retentionPeriod        = 2
)

type Storage struct {
	*bolt.DB
	account string
}

// NewStorage initializes and returns a new Storage instance, ensuring that the bucket for the given account is created.
func NewStorage(account string, path string) (*Storage, error) {
	dbPath := filepath.Join(path, "sdm-sources.db")
	log.Debug().Msgf("Opening database at %s", dbPath)
	db, err := bolt.Open(dbPath, 0o600, nil)
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

	storage.removeOldBuckets(retentionPeriod)
	return storage, nil
}

// ensureBucketExists ensures that the bucket for the given account exists in the database.
func (s *Storage) ensureBucketExists() error {
	bucketKey := buildBucketKey(s.account, currentDBVersion)
	return s.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketKey)
		if err != nil {
			return fmt.Errorf("failed to create bucket for account %s with key %s: %w", s.account, bucketKey, err)
		}
		return nil
	})
}

// buildBucketKey constructs a bucket key using the account and the database version.
func buildBucketKey(account string, version int) []byte {
	return []byte(fmt.Sprintf("%s:%s:v%d", account, datasourceBucketPrefix, version))
}

// // StoreServers stores the provided datasources for the specified account.
// func (s *Storage) StoreServers(datasources []DataSource) error {
// 	bucketKey := buildBucketKey(s.account, currentDBVersion)
// 	return s.Update(func(tx *bolt.Tx) error {
// 		log.Debug().Msgf("Storing %d datasources", len(datasources))
// 		bucket := tx.Bucket(bucketKey)
// 		if bucket == nil {
// 			return fmt.Errorf("bucket for account %s not found", s.account)
// 		}
// 		for _, ds := range datasources {
// 			// Encode the DataSource and handle any errors
// 			encodedData, err := ds.Encode()
// 			if err != nil {
// 				return fmt.Errorf("failed to encode datasource %s: %w", ds.Name, err)
// 			}
// 			// Store the encoded DataSource in the bucket
// 			if err := bucket.Put(ds.Key(), encodedData); err != nil {
// 				return fmt.Errorf("failed to store datasource %s: %w", ds.Name, err)
// 			}
// 		}
// 		log.Debug().Msgf("Successfully stored %d datasources", len(datasources))
// 		return nil
// 	})
// }

// StoreServers stores the provided datasources for the specified account.
func (s *Storage) StoreServers(datasources []DataSource) error {
	bucketKey := buildBucketKey(s.account, currentDBVersion)
	return s.Update(func(tx *bolt.Tx) error {
		log.Debug().Msgf("Storing %d datasources", len(datasources))
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return fmt.Errorf("bucket for account %s not found", s.account)
		}
		for _, ds := range datasources {
			// Retrieve the existing DataSource from the database
			existingData := bucket.Get(ds.Key())
			if existingData != nil {
				var existingDS DataSource
				if err := existingDS.Decode(existingData); err != nil {
					return fmt.Errorf("failed to decode existing datasource %s: %w", ds.Name, err)
				}
				// Update the LRU value of the existing DataSource
				ds.LRU = existingDS.LRU
			}

			// Encode the updated DataSource and handle any errors
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
	bucketKey := buildBucketKey(s.account, currentDBVersion)
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
	bucketKey := buildBucketKey(s.account, currentDBVersion)
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

func (s *Storage) UpdateLastUsed(ds DataSource) error {
	ds.LRU = time.Now().Unix()
	bucketKey := buildBucketKey(s.account, currentDBVersion)
	return s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return fmt.Errorf("bucket for account %s not found", s.account)
		}

		// Encode the DataSource and handle any errors
		encodedData, err := ds.Encode()
		if err != nil {
			return fmt.Errorf("failed to encode datasource %s: %w", ds.Name, err)
		}

		// Store the encoded DataSource in the bucket
		if err := bucket.Put(ds.Key(), encodedData); err != nil {
			return fmt.Errorf("failed to store datasource %s: %w", ds.Name, err)
		}

		return nil
	})
}

// removeOldBuckets removes old buckets that are older than the specified retention period.
func (s *Storage) removeOldBuckets(retentionPeriod int) error {
	return s.Update(func(tx *bolt.Tx) error {
		c := tx.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			parts := strings.Split(string(k), ":")

			if len(parts) == 2 {
				log.Debug().Msgf("Removing bucket without version: %s", k)
				if err := tx.DeleteBucket([]byte(k)); err != nil {
					return fmt.Errorf("failed to delete bucket %s: %w", k, err)
				}
				continue
			} else if len(parts) == 3 {
				var version int
				// Bucket has a version
				var err error
				version, err = strconv.Atoi(parts[len(parts)-1][1:])
				if err != nil {
					log.Error().Msgf("Failed to parse version from bucket: %s, error: %v", k, err)
					continue
				}

				if currentDBVersion-version > retentionPeriod {
					log.Debug().Msgf("Removing old bucket: %s", k)
					if err := tx.DeleteBucket([]byte(k)); err != nil {
						return fmt.Errorf("failed to delete bucket %s: %w", k, err)
					}
				}
			} else {
				log.Error().Msgf("Failed to parse bucket: %s", k)
				continue
			}

		}
		return nil
	})
}

func (s *Storage) Wipe() error {
	return s.Update(func(tx *bolt.Tx) error {
		c := tx.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			log.Debug().Msgf("Removing bucket: %s", k)
			if strings.HasPrefix(string(k), s.account) {
				if err := tx.DeleteBucket([]byte(k)); err != nil {
					return fmt.Errorf("failed to delete bucket %s: %w", k, err)
				}
			}
		}
		return nil
	})
}
