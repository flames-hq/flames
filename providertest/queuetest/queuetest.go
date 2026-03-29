package queuetest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flames-hq/flames/provider/providererr"
	"github.com/flames-hq/flames/provider/queue"
)

func Run(t *testing.T, newQueue func() queue.WorkQueue) {
	t.Run("EnqueueAndDequeue", func(t *testing.T) { testEnqueueAndDequeue(t, newQueue()) })
	t.Run("NoJobs", func(t *testing.T) { testNoJobs(t, newQueue()) })
	t.Run("Ack", func(t *testing.T) { testAck(t, newQueue()) })
	t.Run("Nack", func(t *testing.T) { testNack(t, newQueue()) })
	t.Run("LeaseExpiry", func(t *testing.T) { testLeaseExpiry(t, newQueue()) })
	t.Run("MultiTopic", func(t *testing.T) { testMultiTopic(t, newQueue()) })
}

func testEnqueueAndDequeue(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()
	payload := []byte(`{"action":"test"}`)

	id, err := q.Enqueue(ctx, "tasks", payload)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if id == "" {
		t.Fatal("Enqueue returned empty ID")
	}

	job, err := q.Dequeue(ctx, "tasks", 5*time.Second)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if job.ID != id {
		t.Errorf("got job ID %q, want %q", job.ID, id)
	}
	if !bytes.Equal(job.Payload, payload) {
		t.Errorf("got payload %q, want %q", job.Payload, payload)
	}
	if job.Topic != "tasks" {
		t.Errorf("got topic %q, want %q", job.Topic, "tasks")
	}
	if job.DequeueCount != 1 {
		t.Errorf("got DequeueCount %d, want 1", job.DequeueCount)
	}
}

func testNoJobs(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()

	_, err := q.Dequeue(ctx, "empty-topic", 5*time.Second)
	if !errors.Is(err, providererr.ErrNoJobs) {
		t.Errorf("expected ErrNoJobs, got %v", err)
	}
}

func testAck(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()

	id, _ := q.Enqueue(ctx, "ack-topic", []byte("data"))
	_, _ = q.Dequeue(ctx, "ack-topic", 5*time.Second)

	if err := q.Ack(ctx, id); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	// Queue should now be empty.
	_, err := q.Dequeue(ctx, "ack-topic", 5*time.Second)
	if !errors.Is(err, providererr.ErrNoJobs) {
		t.Errorf("expected ErrNoJobs after ack, got %v", err)
	}
}

func testNack(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()

	id, _ := q.Enqueue(ctx, "nack-topic", []byte("data"))
	_, _ = q.Dequeue(ctx, "nack-topic", 5*time.Second)

	// Nack with immediate retry.
	if err := q.Nack(ctx, id, time.Now()); err != nil {
		t.Fatalf("Nack: %v", err)
	}

	// Should be available again immediately.
	job, err := q.Dequeue(ctx, "nack-topic", 5*time.Second)
	if err != nil {
		t.Fatalf("Dequeue after nack: %v", err)
	}
	if job.ID != id {
		t.Errorf("got job ID %q, want %q", job.ID, id)
	}
	if job.DequeueCount != 2 {
		t.Errorf("got DequeueCount %d, want 2", job.DequeueCount)
	}
}

func testLeaseExpiry(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()

	id, _ := q.Enqueue(ctx, "lease-topic", []byte("data"))

	// Dequeue with very short lease.
	_, err := q.Dequeue(ctx, "lease-topic", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	// Job should not be available during lease.
	_, err = q.Dequeue(ctx, "lease-topic", 5*time.Second)
	if !errors.Is(err, providererr.ErrNoJobs) {
		t.Errorf("expected ErrNoJobs during lease, got %v", err)
	}

	// Wait for lease to expire.
	time.Sleep(100 * time.Millisecond)

	// Job should be available again.
	job, err := q.Dequeue(ctx, "lease-topic", 5*time.Second)
	if err != nil {
		t.Fatalf("Dequeue after lease expiry: %v", err)
	}
	if job.ID != id {
		t.Errorf("got job ID %q, want %q", job.ID, id)
	}
	if job.DequeueCount != 2 {
		t.Errorf("got DequeueCount %d, want 2", job.DequeueCount)
	}
}

func testMultiTopic(t *testing.T, q queue.WorkQueue) {
	ctx := context.Background()

	id1, _ := q.Enqueue(ctx, "topic-a", []byte("a"))
	id2, _ := q.Enqueue(ctx, "topic-b", []byte("b"))

	jobA, err := q.Dequeue(ctx, "topic-a", 5*time.Second)
	if err != nil {
		t.Fatalf("Dequeue topic-a: %v", err)
	}
	if jobA.ID != id1 {
		t.Errorf("topic-a: got ID %q, want %q", jobA.ID, id1)
	}

	jobB, err := q.Dequeue(ctx, "topic-b", 5*time.Second)
	if err != nil {
		t.Fatalf("Dequeue topic-b: %v", err)
	}
	if jobB.ID != id2 {
		t.Errorf("topic-b: got ID %q, want %q", jobB.ID, id2)
	}
}
