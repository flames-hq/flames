package cachetest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flames-hq/flames/provider/cache"
	"github.com/flames-hq/flames/provider/providererr"
)

func Run(t *testing.T, newStore func() cache.CacheStore) {
	t.Run("SetAndGet", func(t *testing.T) { testSetAndGet(t, newStore()) })
	t.Run("CacheMiss", func(t *testing.T) { testCacheMiss(t, newStore()) })
	t.Run("TTLExpiry", func(t *testing.T) { testTTLExpiry(t, newStore()) })
	t.Run("Delete", func(t *testing.T) { testDelete(t, newStore()) })
	t.Run("Overwrite", func(t *testing.T) { testOverwrite(t, newStore()) })
}

func RunAtomic(t *testing.T, newStore func() cache.AtomicCacheStore) {
	t.Run("CompareAndSet", func(t *testing.T) { testCompareAndSet(t, newStore()) })
	t.Run("CompareAndSetMismatch", func(t *testing.T) { testCompareAndSetMismatch(t, newStore()) })
}

func testSetAndGet(t *testing.T, s cache.CacheStore) {
	ctx := context.Background()
	val := []byte("hello")

	if err := s.Set(ctx, "key1", val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("got %q, want %q", got, val)
	}
}

func testCacheMiss(t *testing.T, s cache.CacheStore) {
	ctx := context.Background()

	_, err := s.Get(ctx, "nonexistent")
	if !errors.Is(err, providererr.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func testTTLExpiry(t *testing.T, s cache.CacheStore) {
	ctx := context.Background()
	val := []byte("ephemeral")

	if err := s.Set(ctx, "ttl-key", val, 50*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Should be available immediately.
	got, err := s.Get(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("got %q, want %q", got, val)
	}

	// Wait for expiry.
	time.Sleep(100 * time.Millisecond)

	_, err = s.Get(ctx, "ttl-key")
	if !errors.Is(err, providererr.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after TTL expiry, got %v", err)
	}
}

func testDelete(t *testing.T, s cache.CacheStore) {
	ctx := context.Background()
	_ = s.Set(ctx, "del-key", []byte("data"), 0)

	if err := s.Delete(ctx, "del-key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(ctx, "del-key")
	if !errors.Is(err, providererr.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func testOverwrite(t *testing.T, s cache.CacheStore) {
	ctx := context.Background()
	_ = s.Set(ctx, "ow-key", []byte("v1"), 0)
	_ = s.Set(ctx, "ow-key", []byte("v2"), 0)

	got, err := s.Get(ctx, "ow-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, []byte("v2")) {
		t.Errorf("got %q, want %q", got, "v2")
	}
}

func testCompareAndSet(t *testing.T, s cache.AtomicCacheStore) {
	ctx := context.Background()
	_ = s.Set(ctx, "cas-key", []byte("old"), 0)

	ok, err := s.CompareAndSet(ctx, "cas-key", []byte("old"), []byte("new"), 0)
	if err != nil {
		t.Fatalf("CompareAndSet: %v", err)
	}
	if !ok {
		t.Error("CompareAndSet should have succeeded")
	}

	got, _ := s.Get(ctx, "cas-key")
	if !bytes.Equal(got, []byte("new")) {
		t.Errorf("got %q, want %q", got, "new")
	}
}

func testCompareAndSetMismatch(t *testing.T, s cache.AtomicCacheStore) {
	ctx := context.Background()
	_ = s.Set(ctx, "cas-key", []byte("current"), 0)

	ok, err := s.CompareAndSet(ctx, "cas-key", []byte("wrong"), []byte("new"), 0)
	if err != nil {
		t.Fatalf("CompareAndSet: %v", err)
	}
	if ok {
		t.Error("CompareAndSet should have failed with mismatched old value")
	}

	got, _ := s.Get(ctx, "cas-key")
	if !bytes.Equal(got, []byte("current")) {
		t.Errorf("value should not have changed, got %q", got)
	}
}
