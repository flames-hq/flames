// Package queue defines the WorkQueue interface for at-least-once background job processing.
package queue

import (
	"context"
	"time"

	"github.com/flames-hq/flames/model"
)

// WorkQueue provides at-least-once delivery for background jobs. Dequeue is non-blocking;
// unacknowledged jobs are automatically redelivered after their lease expires.
type WorkQueue interface {
	Enqueue(ctx context.Context, topic string, payload []byte) (string, error)
	Dequeue(ctx context.Context, topic string, leaseTimeout time.Duration) (model.Job, error)
	Ack(ctx context.Context, jobID string) error
	Nack(ctx context.Context, jobID string, retryAt time.Time) error
}
