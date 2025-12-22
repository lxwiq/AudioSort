package cache

import (
	"encoding/json"
	"time"

	"audiosort/pkg/models"

	"go.etcd.io/bbolt"
)

type Store struct {
	db *bbolt.DB
}

type CachedBook struct {
	Query     string              `json:"query"`
	Metadata  models.BookMetadata `json:"metadata"`
	FetchedAt time.Time           `json:"fetched_at"`
}

type ProcessedBook struct {
	SourcePath  string    `json:"source_path"`
	DestPath    string    `json:"dest_path"`
	ProcessedAt time.Time `json:"processed_at"`
	Checksum    string    `json:"checksum"`
}

var (
	metadataBucket  = []byte("metadata")
	processedBucket = []byte("processed")
)

func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(metadataBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(processedBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) GetMetadata(query string) (*CachedBook, bool) {
	var cached CachedBook

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(metadataBucket)
		if b == nil {
			return nil
		}
		data := b.Get([]byte(query))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &cached)
	})

	if err != nil || cached.Query == "" {
		return nil, false
	}

	if time.Since(cached.FetchedAt) > 30*24*time.Hour {
		return nil, false
	}

	return &cached, true
}

func (s *Store) SetMetadata(query string, metadata models.BookMetadata) error {
	cached := CachedBook{
		Query:     query,
		Metadata:  metadata,
		FetchedAt: time.Now(),
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(metadataBucket)
		if b == nil {
			return nil
		}
		data, err := json.Marshal(cached)
		if err != nil {
			return err
		}
		return b.Put([]byte(query), data)
	})
}

func (s *Store) IsProcessed(path string) bool {
	var exists bool

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(processedBucket)
		if b == nil {
			return nil
		}
		exists = b.Get([]byte(path)) != nil
		return nil
	})

	return exists && err == nil
}

func (s *Store) MarkProcessed(sourcePath, destPath, checksum string) error {
	processed := ProcessedBook{
		SourcePath:  sourcePath,
		DestPath:    destPath,
		ProcessedAt: time.Now(),
		Checksum:    checksum,
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(processedBucket)
		if b == nil {
			return nil
		}
		data, err := json.Marshal(processed)
		if err != nil {
			return err
		}
		return b.Put([]byte(sourcePath), data)
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}
