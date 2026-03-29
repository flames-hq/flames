package model

import "time"

// BlobMeta holds metadata about a stored blob, including its key, size, SHA-256 checksum,
// content type, and arbitrary user-defined key-value metadata.
type BlobMeta struct {
	Key         string            `json:"key"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
}
