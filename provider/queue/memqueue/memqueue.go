// Package memqueue provides an in-memory WorkQueue implementation for development mode.
// Lease expiry is checked at dequeue time rather than by a background goroutine.
package memqueue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/providererr"
	"github.com/flames-hq/flames/provider/queue"
)

var _ queue.WorkQueue = (*Queue)(nil)

type queueEntry struct {
	job model.Job
}

// Queue is a mutex-protected in-memory WorkQueue with per-topic job slices and lease tracking.
type Queue struct {
	mu     sync.Mutex
	topics map[string][]queueEntry
	index  map[string]topicRef // jobID -> topic + index for fast lookup
}

type topicRef struct {
	topic string
}

// New creates a new in-memory WorkQueue.
func New() *Queue {
	return &Queue{
		topics: make(map[string][]queueEntry),
		index:  make(map[string]topicRef),
	}
}

// Enqueue adds a new job to the given topic with a generated ID and returns it.
func (q *Queue) Enqueue(_ context.Context, topic string, payload []byte) (string, error) {
	id := newID()
	p := make([]byte, len(payload))
	copy(p, payload)

	job := model.Job{
		ID:         id,
		Topic:      topic,
		Payload:    p,
		EnqueuedAt: time.Now(),
	}

	q.mu.Lock()
	q.topics[topic] = append(q.topics[topic], queueEntry{job: job})
	q.index[id] = topicRef{topic: topic}
	q.mu.Unlock()

	return id, nil
}

// Dequeue returns the first available job from the topic whose lease has expired or was never set.
// Returns ErrNoJobs if no jobs are available. Does not block.
func (q *Queue) Dequeue(_ context.Context, topic string, leaseTimeout time.Duration) (model.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	entries := q.topics[topic]
	now := time.Now()

	for i := range entries {
		e := &entries[i]
		if e.job.LeaseExpiresAt.IsZero() || now.After(e.job.LeaseExpiresAt) {
			e.job.LeaseExpiresAt = now.Add(leaseTimeout)
			e.job.DequeueCount++
			q.topics[topic] = entries
			return e.job, nil
		}
	}

	return model.Job{}, providererr.ErrNoJobs
}

// Ack acknowledges a job, removing it from the queue permanently.
func (q *Queue) Ack(_ context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	ref, ok := q.index[jobID]
	if !ok {
		return providererr.NotFound("job", jobID)
	}

	entries := q.topics[ref.topic]
	for i, e := range entries {
		if e.job.ID == jobID {
			q.topics[ref.topic] = append(entries[:i], entries[i+1:]...)
			delete(q.index, jobID)
			return nil
		}
	}

	return providererr.NotFound("job", jobID)
}

// Nack negatively acknowledges a job, making it available for redelivery at the specified retryAt time.
func (q *Queue) Nack(_ context.Context, jobID string, retryAt time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	ref, ok := q.index[jobID]
	if !ok {
		return providererr.NotFound("job", jobID)
	}

	entries := q.topics[ref.topic]
	for i := range entries {
		if entries[i].job.ID == jobID {
			entries[i].job.LeaseExpiresAt = retryAt
			q.topics[ref.topic] = entries
			return nil
		}
	}

	return providererr.NotFound("job", jobID)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
