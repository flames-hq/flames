package httpapi

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"sync"
	"time"
)

const idempotencyTTL = 24 * time.Hour

type idempotencyEntry struct {
	requestHash [32]byte
	statusCode  int
	body        []byte
	createdAt   time.Time
}

type idempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]idempotencyEntry
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{
		entries: make(map[string]idempotencyEntry),
	}
}

// wrap returns an http.HandlerFunc that enforces idempotency on the given handler.
func (s *idempotencyStore) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		hash := sha256.Sum256(body)

		s.mu.RLock()
		entry, exists := s.entries[key]
		s.mu.RUnlock()

		if exists {
			// Lazy eviction: treat expired entries as nonexistent.
			if time.Since(entry.createdAt) > idempotencyTTL {
				s.mu.Lock()
				delete(s.entries, key)
				s.mu.Unlock()
				exists = false
			}
		}

		if exists {
			if entry.requestHash != hash {
				writeErrorMessage(w, http.StatusConflict, "conflict", "idempotency key reused with different request body")
				return
			}
			// Replay cached response.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(entry.statusCode)
			w.Write(entry.body)
			return
		}

		// Execute and capture the response.
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)

		s.mu.Lock()
		s.entries[key] = idempotencyEntry{
			requestHash: hash,
			statusCode:  rec.statusCode,
			body:        rec.body.Bytes(),
			createdAt:   time.Now(),
		}
		s.mu.Unlock()
	}
}

// responseRecorder captures the status code and body written by a handler.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
