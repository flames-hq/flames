package model

import "time"

// Job represents a unit of work in the WorkQueue. It carries an opaque payload, belongs to a
// topic, and tracks lease expiry for at-least-once delivery semantics.
type Job struct {
	ID             string    `json:"id"`
	Topic          string    `json:"topic"`
	Payload        []byte    `json:"payload"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
	DequeueCount   int       `json:"dequeue_count"`
}
