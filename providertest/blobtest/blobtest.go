package blobtest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/blob"
	"github.com/flames-hq/flames/provider/providererr"
)

func Run(t *testing.T, newStore func() blob.BlobStore) {
	t.Run("PutAndGet", func(t *testing.T) { testPutAndGet(t, newStore()) })
	t.Run("Head", func(t *testing.T) { testHead(t, newStore()) })
	t.Run("Delete", func(t *testing.T) { testDelete(t, newStore()) })
	t.Run("Exists", func(t *testing.T) { testExists(t, newStore()) })
	t.Run("List", func(t *testing.T) { testList(t, newStore()) })
	t.Run("NotFoundErrors", func(t *testing.T) { testNotFoundErrors(t, newStore()) })
}

func testPutAndGet(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()
	data := []byte("hello, blobs!")
	meta := model.BlobMeta{ContentType: "text/plain"}

	if err := s.Put(ctx, "test/key", bytes.NewReader(data), meta); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := s.Get(ctx, "test/key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func testHead(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()
	data := []byte("head test data")
	meta := model.BlobMeta{ContentType: "application/octet-stream"}

	if err := s.Put(ctx, "head/key", bytes.NewReader(data), meta); err != nil {
		t.Fatalf("Put: %v", err)
	}

	m, err := s.Head(ctx, "head/key")
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if m.Key != "head/key" {
		t.Errorf("got Key %q, want %q", m.Key, "head/key")
	}
	if m.Size != int64(len(data)) {
		t.Errorf("got Size %d, want %d", m.Size, len(data))
	}
	if m.ContentType != "application/octet-stream" {
		t.Errorf("got ContentType %q, want %q", m.ContentType, "application/octet-stream")
	}
	if m.Checksum == "" {
		t.Error("Checksum is empty")
	}
}

func testDelete(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()
	_ = s.Put(ctx, "del/key", bytes.NewReader([]byte("x")), model.BlobMeta{})

	if err := s.Delete(ctx, "del/key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, _ := s.Exists(ctx, "del/key")
	if exists {
		t.Error("blob still exists after delete")
	}
}

func testExists(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()

	exists, err := s.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent key")
	}

	_ = s.Put(ctx, "exists/key", bytes.NewReader([]byte("data")), model.BlobMeta{})

	exists, err = s.Exists(ctx, "exists/key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected true for existing key")
	}
}

func testList(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()
	_ = s.Put(ctx, "images/a.png", bytes.NewReader([]byte("a")), model.BlobMeta{})
	_ = s.Put(ctx, "images/b.png", bytes.NewReader([]byte("b")), model.BlobMeta{})
	_ = s.Put(ctx, "docs/c.txt", bytes.NewReader([]byte("c")), model.BlobMeta{})

	list, err := s.List(ctx, "images/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 blobs with prefix images/, got %d", len(list))
	}

	all, err := s.List(ctx, "")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total blobs, got %d", len(all))
	}
}

func testNotFoundErrors(t *testing.T, s blob.BlobStore) {
	ctx := context.Background()

	_, err := s.Get(ctx, "missing")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("Get: expected ErrNotFound, got %v", err)
	}

	_, err = s.Head(ctx, "missing")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("Head: expected ErrNotFound, got %v", err)
	}

	err = s.Delete(ctx, "missing")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("Delete: expected ErrNotFound, got %v", err)
	}
}
