// Package memblob provides an in-memory BlobStore implementation for development mode.
package memblob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/blob"
	"github.com/flames-hq/flames/provider/providererr"
)

var _ blob.BlobStore = (*Store)(nil)

type storedBlob struct {
	data []byte
	meta model.BlobMeta
}

type Store struct {
	mu    sync.RWMutex
	blobs map[string]storedBlob
}

// New creates a new in-memory BlobStore.
func New() *Store {
	return &Store{
		blobs: make(map[string]storedBlob),
	}
}

// Put reads all data from r and stores it under the given key, computing the SHA-256 checksum and size automatically.
func (s *Store) Put(_ context.Context, key string, r io.Reader, meta model.BlobMeta) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	h := sha256.Sum256(data)
	meta.Key = key
	meta.Size = int64(len(data))
	meta.Checksum = hex.EncodeToString(h[:])
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}

	s.mu.Lock()
	s.blobs[key] = storedBlob{data: data, meta: meta}
	s.mu.Unlock()

	return nil
}

// Get returns the blob data as an io.ReadCloser. Returns ErrNotFound if the key does not exist.
func (s *Store) Get(_ context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	b, ok := s.blobs[key]
	s.mu.RUnlock()

	if !ok {
		return nil, providererr.NotFound("blob", key)
	}
	return io.NopCloser(bytes.NewReader(b.data)), nil
}

// Head returns the metadata for a blob without its data. Returns ErrNotFound if the key does not exist.
func (s *Store) Head(_ context.Context, key string) (model.BlobMeta, error) {
	s.mu.RLock()
	b, ok := s.blobs[key]
	s.mu.RUnlock()

	if !ok {
		return model.BlobMeta{}, providererr.NotFound("blob", key)
	}
	return b.meta, nil
}

// Delete removes a blob by key. Returns ErrNotFound if the key does not exist.
func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blobs[key]; !ok {
		return providererr.NotFound("blob", key)
	}
	delete(s.blobs, key)
	return nil
}

// Exists reports whether a blob with the given key exists.
func (s *Store) Exists(_ context.Context, key string) (bool, error) {
	s.mu.RLock()
	_, ok := s.blobs[key]
	s.mu.RUnlock()
	return ok, nil
}

// List returns metadata for all blobs whose keys match the given prefix.
func (s *Store) List(_ context.Context, prefix string) ([]model.BlobMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.BlobMeta
	for k, b := range s.blobs {
		if strings.HasPrefix(k, prefix) {
			result = append(result, b.meta)
		}
	}
	return result, nil
}
