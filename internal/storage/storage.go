package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

// Database constants
const (
	datasourceBucketPrefix = "datasource"
	currentDBVersion       = 2 // increment this whenever the database schema changes
	retentionPeriod        = 2
	defaultTimeout         = 5 * time.Second
)

// Common errors
var (
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrDataSourceNotFound = errors.New("datasource not found")
	ErrDatabaseClosed     = errors.New("database is closed")
)

// Storage manages persistence of data sources using BoltDB
type Storage struct {
	*bolt.DB
	account string
	timeout time.Duration
}

// StorageOption is a function option for configuring the Storage
type StorageOption func(*Storage)

// WithTimeout sets a custom timeout for database operations
func WithTimeout(timeout time.Duration) StorageOption {
	return func(s *Storage) {
		s.timeout = timeout
	}
}

// NewStorage initializes and returns a new Storage instance
func NewStorage(account string, path string, opts ...StorageOption) (*Storage, error) {
	if account == "" {
		return nil, errors.New("account cannot be empty")
	}

	dbPath := filepath.Join(path, "sdm-sources.db")
	log.Debug().Str("path", dbPath).Msg("Opening database")

	// Open database with options
	options := &bolt.Options{
		Timeout: defaultTimeout,
	}

	db, err := bolt.Open(dbPath, 0o600, options)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{
		DB:      db,
		account: account,
		timeout: defaultTimeout,
	}

	// Apply options
	for _, opt := range opts {
		opt(storage)
	}

	// Initialize bucket
	if err := storage.ensureBucketExists(); err != nil {
		// Close DB if initialization fails
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Perform maintenance
	if err := storage.removeOldBuckets(retentionPeriod); err != nil {
		log.Warn().Err(err).Msg("Failed to remove old buckets during initialization")
	}

	return storage, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.DB == nil {
		return ErrDatabaseClosed
	}

	log.Debug().Msg("Closing database connection")
	return s.DB.Close()
}

// ensureBucketExists ensures that the bucket for the account exists
func (s *Storage) ensureBucketExists() error {
	bucketKey := buildBucketKey(s.account, currentDBVersion)
	log.Debug().Str("bucket", string(bucketKey)).Msg("Ensuring bucket exists")

	return s.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketKey)
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		return nil
	})
}

// buildBucketKey constructs a bucket key
func buildBucketKey(account string, version int) []byte {
	return []byte(fmt.Sprintf("%s:%s:v%d", account, datasourceBucketPrefix, version))
}

// StoreServers stores the provided datasources
func (s *Storage) StoreServers(datasources []DataSource) error {
	if len(datasources) == 0 {
		log.Debug().Msg("No datasources to store")
		return nil
	}

	bucketKey := buildBucketKey(s.account, currentDBVersion)
	log.Debug().
		Int("count", len(datasources)).
		Str("bucket", string(bucketKey)).
		Msg("Storing datasources")

	return s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return ErrBucketNotFound
		}

		successCount := 0
		for _, ds := range datasources {
			// Preserve existing LRU value if present
			existingData := bucket.Get(ds.Key())
			if existingData != nil {
				var existingDS DataSource
				if err := existingDS.Decode(existingData); err != nil {
					log.Warn().
						Err(err).
						Str("name", ds.Name).
						Msg("Failed to decode existing datasource")
				} else {
					ds.LRU = existingDS.LRU
				}
			}

			// Encode and store the datasource
			encodedData, err := ds.Encode()
			if err != nil {
				log.Error().
					Err(err).
					Str("name", ds.Name).
					Msg("Failed to encode datasource")
				continue
			}

			if err := bucket.Put(ds.Key(), encodedData); err != nil {
				log.Error().
					Err(err).
					Str("name", ds.Name).
					Msg("Failed to store datasource")
				continue
			}

			successCount++
		}

		log.Debug().
			Int("total", len(datasources)).
			Int("success", successCount).
			Msg("Stored datasources")

		return nil
	})
}

// RetrieveDatasources retrieves all datasources
func (s *Storage) RetrieveDatasources() ([]DataSource, error) {
	bucketKey := buildBucketKey(s.account, currentDBVersion)
	log.Debug().Str("bucket", string(bucketKey)).Msg("Retrieving datasources")

	var datasources []DataSource

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return ErrBucketNotFound
		}

		// Pre-allocate with reasonable capacity
		datasources = make([]DataSource, 0, 100)

		// Iterate through all entries
		return bucket.ForEach(func(k, v []byte) error {
			var ds DataSource
			if err := ds.Decode(v); err != nil {
				log.Warn().
					Err(err).
					Str("key", string(k)).
					Msg("Failed to decode datasource")
				return nil // Continue despite error
			}

			datasources = append(datasources, ds)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Int("count", len(datasources)).Msg("Retrieved datasources")
	return datasources, nil
}

// GetDatasource retrieves a single datasource by name
func (s *Storage) GetDatasource(name string) (DataSource, error) {
	if name == "" {
		return DataSource{}, fmt.Errorf("datasource name cannot be empty")
	}

	bucketKey := buildBucketKey(s.account, currentDBVersion)
	log.Debug().
		Str("name", name).
		Str("bucket", string(bucketKey)).
		Msg("Retrieving datasource")

	var datasource DataSource

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return ErrBucketNotFound
		}

		value := bucket.Get([]byte(name))
		if value == nil {
			return ErrDataSourceNotFound
		}

		if err := datasource.Decode(value); err != nil {
			return fmt.Errorf("failed to decode datasource: %w", err)
		}

		return nil
	})
	if err != nil {
		log.Debug().
			Err(err).
			Str("name", name).
			Msg("Failed to retrieve datasource")
		return DataSource{}, err
	}

	return datasource, nil
}

// UpdateLastUsed updates the last used timestamp of a datasource
func (s *Storage) UpdateLastUsed(ds DataSource) error {
	if ds.Name == "" {
		return fmt.Errorf("datasource name cannot be empty")
	}

	// Update timestamp
	ds.LRU = time.Now().Unix()

	bucketKey := buildBucketKey(s.account, currentDBVersion)
	log.Debug().
		Str("name", ds.Name).
		Int64("timestamp", ds.LRU).
		Msg("Updating last used timestamp")

	return s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKey)
		if bucket == nil {
			return ErrBucketNotFound
		}

		// Encode and store
		encodedData, err := ds.Encode()
		if err != nil {
			return fmt.Errorf("failed to encode datasource: %w", err)
		}

		if err := bucket.Put(ds.Key(), encodedData); err != nil {
			return fmt.Errorf("failed to store datasource: %w", err)
		}

		return nil
	})
}

// removeOldBuckets removes buckets older than the retention period
func (s *Storage) removeOldBuckets(retentionPeriod int) error {
	log.Debug().Int("retention_period", retentionPeriod).Msg("Removing old buckets")

	return s.Update(func(tx *bolt.Tx) error {
		var bucketsToDelete [][]byte

		// First pass: identify buckets to delete
		err := tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			bucketName := string(name)
			parts := strings.Split(bucketName, ":")

			if len(parts) == 2 {
				// Legacy bucket without version
				log.Debug().Str("bucket", bucketName).Msg("Found legacy bucket to remove")
				bucketsToDelete = append(bucketsToDelete, name)
			} else if len(parts) == 3 && strings.HasPrefix(parts[0], s.account) {
				// Check versioned bucket
				versionStr := parts[2]
				if !strings.HasPrefix(versionStr, "v") {
					log.Warn().Str("bucket", bucketName).Msg("Invalid version format in bucket name")
					return nil
				}

				version, err := strconv.Atoi(versionStr[1:])
				if err != nil {
					log.Warn().
						Err(err).
						Str("bucket", bucketName).
						Msg("Failed to parse version")
					return nil
				}

				if currentDBVersion-version > retentionPeriod {
					log.Debug().
						Str("bucket", bucketName).
						Int("version", version).
						Msg("Found old bucket to remove")
					bucketsToDelete = append(bucketsToDelete, name)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to enumerate buckets: %w", err)
		}

		// Second pass: delete the identified buckets
		for _, name := range bucketsToDelete {
			log.Debug().Str("bucket", string(name)).Msg("Removing bucket")
			if err := tx.DeleteBucket(name); err != nil {
				log.Warn().
					Err(err).
					Str("bucket", string(name)).
					Msg("Failed to delete bucket")
				// Continue with other buckets
			}
		}

		log.Debug().Int("count", len(bucketsToDelete)).Msg("Removed old buckets")
		return nil
	})
}

// Wipe removes all buckets for the current account
func (s *Storage) Wipe() error {
	log.Debug().Str("account", s.account).Msg("Wiping database for account")

	return s.Update(func(tx *bolt.Tx) error {
		var bucketsToDelete [][]byte

		// First pass: identify buckets to delete
		err := tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			if strings.HasPrefix(string(name), s.account) {
				bucketsToDelete = append(bucketsToDelete, name)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to enumerate buckets: %w", err)
		}

		// Second pass: delete the identified buckets
		for _, name := range bucketsToDelete {
			log.Debug().Str("bucket", string(name)).Msg("Removing bucket")
			if err := tx.DeleteBucket(name); err != nil {
				return fmt.Errorf("failed to delete bucket %s: %w", string(name), err)
			}
		}

		log.Debug().Int("count", len(bucketsToDelete)).Msg("Wiped buckets for account")
		return nil
	})
}
