// Package memcache provides an in-memory CacheStore implementation for development mode.
// Expired entries are lazily evicted on access rather than by a background reaper.
package memcache

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/flames-hq/flames/provider/cache"
	"github.com/flames-hq/flames/provider/providererr"
)

var (
	_ cache.CacheStore       = (*Store)(nil)
	_ cache.AtomicCacheStore = (*Store)(nil)
)

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// Store is a mutex-protected in-memory CacheStore that implements both CacheStore and AtomicCacheStore.
type Store struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// New creates a new in-memory CacheStore.
func New() *Store {
	return &Store{
		entries: make(map[string]cacheEntry),
	}
}

// Get returns the cached value for the given key. Returns ErrCacheMiss if the key does not exist or has expired.
func (s *Store) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok {
		return nil, providererr.ErrCacheMiss
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, providererr.ErrCacheMiss
	}
	return entry.value, nil
}

// Set stores a value with the given TTL. A TTL of zero means the entry never expires.
func (s *Store) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	entry := cacheEntry{
		value: make([]byte, len(value)),
	}
	copy(entry.value, value)
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	s.entries[key] = entry
	s.mu.Unlock()

	return nil
}

// Delete removes a cached entry by key. No error is returned if the key does not exist.
func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
	return nil
}

// CompareAndSet atomically replaces the value if it matches old. Returns true if the swap succeeded.
func (s *Store) CompareAndSet(_ context.Context, key string, old, new []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok {
		if old != nil {
			return false, nil
		}
	} else {
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			delete(s.entries, key)
			if old != nil {
				return false, nil
			}
		} else if !bytes.Equal(entry.value, old) {
			return false, nil
		}
	}

	newEntry := cacheEntry{
		value: make([]byte, len(new)),
	}
	copy(newEntry.value, new)
	if ttl > 0 {
		newEntry.expiresAt = time.Now().Add(ttl)
	}
	s.entries[key] = newEntry
	return true, nil
}
