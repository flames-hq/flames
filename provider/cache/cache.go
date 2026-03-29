// Package cache defines the CacheStore interface for ephemeral key-value caching.
package cache

import (
	"context"
	"time"
)

// CacheStore provides TTL-based ephemeral caching. The control plane must not depend on
// CacheStore for correctness — the system must function identically if every call returns a miss.
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// AtomicCacheStore extends CacheStore with a compare-and-set operation for implementations that support it.
type AtomicCacheStore interface {
	CacheStore
	CompareAndSet(ctx context.Context, key string, old, new []byte, ttl time.Duration) (bool, error)
}
