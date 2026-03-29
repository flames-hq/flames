// Package blob defines the BlobStore interface for storing and retrieving opaque artifacts.
package blob

import (
	"context"
	"io"

	"github.com/flames-hq/flames/model"
)

// BlobStore provides key-based storage for opaque binary artifacts.
// Keys are flat strings with no directory semantics; prefix-based listing is supported.
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, meta model.BlobMeta) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (model.BlobMeta, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]model.BlobMeta, error)
}
