---
schema_version: 1
id: lR5hVebvFb
created_at: 2026-03-29T15:46:42Z
updated_at: 2026-03-29T16:14:37Z
labels:
- SPEC-001
---
# Design

## Approach

Define each provider as a single Go interface in its own package, with domain types in a shared `model` package and errors in a shared `providererr` package. Default implementations (in-memory maps + mutexes, no-op for ingress) live in sub-packages adjacent to each interface. Conformance test suites are exported helper functions that any adapter can import and run against its own implementation.

This "interface-per-package" approach was chosen over a single monolithic `provider` interface because it keeps import graphs narrow — a component that only needs `BlobStore` never pulls in `StateStore` types — and it maps cleanly to the future adapter packages (e.g., `provider/state/postgres` imports `provider/state` and `providertest/statetest`).

## Architecture

### Package Layout

```
github.com/flames-hq/flames/
├── go.mod
├── model/                          # Shared domain types (FR-004)
│   ├── vm.go                       # VM, VMSpec, DesiredState, ObservedState enums
│   ├── controller.go               # Controller, Heartbeat
│   ├── event.go                    # Event, EventFilter
│   ├── blob.go                     # BlobMeta
│   ├── job.go                      # Job
│   └── endpoint.go                 # Endpoint, Route, Target
├── provider/
│   ├── providererr/                # Structured error types (FR-005)
│   │   └── errors.go
│   ├── state/                      # StateStore interface (FR-010)
│   │   ├── state.go                # interface definition
│   │   └── memstate/               # In-memory default (FR-013)
│   │       └── memstate.go
│   ├── blob/                       # BlobStore interface (FR-020)
│   │   ├── blob.go
│   │   └── memblob/                # In-memory default (FR-023)
│   │       └── memblob.go
│   ├── cache/                      # CacheStore interface (FR-030)
│   │   ├── cache.go
│   │   └── memcache/               # In-memory default (FR-032)
│   │       └── memcache.go
│   ├── queue/                      # WorkQueue interface (FR-040)
│   │   ├── queue.go
│   │   └── memqueue/               # In-memory default (FR-042)
│   │       └── memqueue.go
│   └── ingress/                    # IngressProvider interface (FR-050)
│       ├── ingress.go
│       └── noop/                   # No-op default (FR-051)
│           └── noop.go
└── providertest/                   # Conformance test suites (FR-070)
    ├── statetest/
    │   └── statetest.go
    ├── blobtest/
    │   └── blobtest.go
    ├── cachetest/
    │   └── cachetest.go
    ├── queuetest/
    │   └── queuetest.go
    └── ingresstest/
        └── ingresstest.go
```

### Dependency Graph

```
model  ←──  provider/providererr  (errors reference model types for metadata)
  ↑              ↑
  │              │
  ├── provider/state/state.go      (interface imports model + providererr)
  ├── provider/blob/blob.go
  ├── provider/cache/cache.go
  ├── provider/queue/queue.go
  └── provider/ingress/ingress.go
        ↑
        │
  provider/*/mem*  or  noop/       (implementations import their parent interface pkg)
        ↑
        │
  providertest/*test/              (test suites import interface pkg + providererr)
```

No circular dependencies. Each implementation imports only its own interface package, `model`, and `providererr`. Conformance tests import the interface package and `providererr` — they never import a specific implementation.

### How It Fits the Architecture

The control-plane components (API server, scheduler, reconciler) will accept these interfaces via constructor injection:

```go
func NewScheduler(state state.StateStore, queue queue.WorkQueue) *Scheduler
```

For local development, `main.go` wires up defaults:

```go
ss := memstate.New()
bs := memblob.New()
cs := memcache.New()
wq := memqueue.New()
ip := noop.New()
```

For hosted deployments, the same constructor signatures accept Postgres/S3/Redis adapters.

## Data Model

### State Enums (`model/vm.go`)

```go
type DesiredState string

const (
    DesiredRunning DesiredState = "running"
    DesiredStopped DesiredState = "stopped"
    DesiredDeleted DesiredState = "deleted"
)

type ObservedState string

const (
    ObservedPending   ObservedState = "pending"
    ObservedScheduled ObservedState = "scheduled"
    ObservedPreparing ObservedState = "preparing"
    ObservedStarting  ObservedState = "starting"
    ObservedRunning   ObservedState = "running"
    ObservedStopping  ObservedState = "stopping"
    ObservedStopped   ObservedState = "stopped"
    ObservedFailed    ObservedState = "failed"
    ObservedDeleted   ObservedState = "deleted"
)
```

### VM (`model/vm.go`)

```go
type VM struct {
    ID            string
    DesiredState  DesiredState
    ObservedState ObservedState
    ImageID       string
    ControllerID  string        // empty until assigned
    Spec          VMSpec
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type VMSpec struct {
    Resources ResourceSpec       `json:"resources"`
    Lifecycle LifecycleSpec      `json:"lifecycle"`
    Runtime   RuntimeSpec        `json:"runtime"`
    Network   NetworkSpec        `json:"network"`
    Storage   StorageSpec        `json:"storage"`
    Placement PlacementSpec      `json:"placement"`
    Metadata  map[string]string  `json:"metadata"`
}
```

`ResourceSpec`, `LifecycleSpec`, etc. are defined as sub-structs with the fields from the architecture (vcpu, memory_mb, timeout_seconds, auto_delete, restart_policy, command, args, env, etc.). Details deferred to implementation — they don't affect interface signatures.

### Controller (`model/controller.go`)

```go
type Controller struct {
    ID              string
    Status          string            // "active", "draining", "offline"
    Labels          map[string]string
    Capacity        CapacityInfo
    LastHeartbeatAt time.Time
}

type CapacityInfo struct {
    TotalVCPUs    int
    TotalMemoryMB int
    UsedVCPUs     int
    UsedMemoryMB  int
}

type Heartbeat struct {
    Status   string
    Capacity CapacityInfo
}
```

### Event (`model/event.go`)

```go
type Event struct {
    ID           string
    VMID         string
    ControllerID string
    Type         string   // "vm.created", "vm.scheduled", etc.
    Payload      []byte   // JSON payload
    CreatedAt    time.Time
}

type EventFilter struct {
    VMID         string
    ControllerID string
    Type         string
    Since        time.Time
    Limit        int
}
```

### BlobMeta (`model/blob.go`)

```go
type BlobMeta struct {
    Key         string
    Size        int64
    Checksum    string            // SHA-256 hex
    ContentType string
    Metadata    map[string]string
    CreatedAt   time.Time
}
```

### Job (`model/job.go`)

```go
type Job struct {
    ID        string
    Topic     string
    Payload   []byte
    LeaseExpiresAt time.Time
    EnqueuedAt     time.Time
    DequeueCount   int
}
```

### Endpoint (`model/endpoint.go`)

```go
type Endpoint struct {
    VMID   string
    Route  Route
    Target Target
}

type Route struct {
    Host string
    Path string
}

type Target struct {
    Address string   // internal IP:port
    Port    int
}
```

## API / Interfaces

### StateStore (`provider/state/state.go`)

```go
package state

import (
    "context"
    "github.com/flames-hq/flames/model"
)

type StateStore interface {
    CreateVM(ctx context.Context, spec model.VMSpec) (string, error)
    GetVM(ctx context.Context, vmID string) (model.VM, error)
    UpdateVMDesiredState(ctx context.Context, vmID string, state model.DesiredState) error
    UpdateVMObservedState(ctx context.Context, vmID string, state model.ObservedState, controllerID string) error
    AssignVM(ctx context.Context, vmID string, controllerID string) error
    ListPendingVMs(ctx context.Context) ([]model.VM, error)

    AppendEvent(ctx context.Context, event model.Event) error
    ListEvents(ctx context.Context, filter model.EventFilter) ([]model.Event, error)

    RegisterController(ctx context.Context, controller model.Controller) error
    UpdateControllerHeartbeat(ctx context.Context, controllerID string, heartbeat model.Heartbeat) error
}
```

### BlobStore (`provider/blob/blob.go`)

```go
package blob

import (
    "context"
    "io"
    "github.com/flames-hq/flames/model"
)

type BlobStore interface {
    Put(ctx context.Context, key string, r io.Reader, meta model.BlobMeta) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Head(ctx context.Context, key string) (model.BlobMeta, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    List(ctx context.Context, prefix string) ([]model.BlobMeta, error)
}
```

`Get` returns `io.ReadCloser` so callers must close the reader. The in-memory implementation returns a `bytes.NewReader` wrapped in `io.NopCloser`. Real adapters (S3, filesystem) return streams that hold resources until closed.

### CacheStore (`provider/cache/cache.go`)

```go
package cache

import (
    "context"
    "time"
)

type CacheStore interface {
    Get(ctx context.Context, key string) ([]byte, error)   // returns providererr.ErrCacheMiss on miss
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

`CompareAndSet` is not on the core interface (FR-031). If needed later, it can be an optional interface:

```go
type AtomicCacheStore interface {
    CacheStore
    CompareAndSet(ctx context.Context, key string, old, new []byte, ttl time.Duration) (bool, error)
}
```

Callers that need CAS can type-assert; others use only `CacheStore`. The in-memory default implements both.

### WorkQueue (`provider/queue/queue.go`)

```go
package queue

import (
    "context"
    "time"
    "github.com/flames-hq/flames/model"
)

type WorkQueue interface {
    Enqueue(ctx context.Context, topic string, payload []byte) (string, error)
    Dequeue(ctx context.Context, topic string, leaseTimeout time.Duration) (model.Job, error)
    Ack(ctx context.Context, jobID string) error
    Nack(ctx context.Context, jobID string, retryAt time.Time) error
}
```

`Dequeue` returns `providererr.ErrNoJobs` when the topic is empty. It does NOT block — polling is the caller's responsibility. This keeps the interface simple and avoids forcing implementations into a particular concurrency model (channels vs. long-poll vs. blocking query).

### IngressProvider (`provider/ingress/ingress.go`)

```go
package ingress

import (
    "context"
    "github.com/flames-hq/flames/model"
)

type IngressProvider interface {
    RegisterEndpoint(ctx context.Context, vmID string, route model.Route, target model.Target) (model.Endpoint, error)
    UnregisterEndpoint(ctx context.Context, vmID string) error
    GetEndpoint(ctx context.Context, vmID string) (model.Endpoint, error)
}
```

### Structured Errors (`provider/providererr/errors.go`)

```go
package providererr

import "fmt"

// Sentinel errors for errors.Is matching.
var (
    ErrNotFound     = &ProviderError{Code: "not_found"}
    ErrAlreadyExists = &ProviderError{Code: "already_exists"}
    ErrConflict     = &ProviderError{Code: "conflict"}
    ErrNotSupported = &ProviderError{Code: "not_supported"}
    ErrCacheMiss    = &ProviderError{Code: "cache_miss"}
    ErrNoJobs       = &ProviderError{Code: "no_jobs"}
)

// ProviderError carries structured metadata about provider failures.
type ProviderError struct {
    Code         string // machine-readable code
    Message      string // human-readable description
    ResourceType string // e.g., "vm", "controller", "blob"
    ResourceID   string // the ID that was looked up
    Err          error  // wrapped cause
}

func (e *ProviderError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    if e.ResourceType != "" && e.ResourceID != "" {
        return fmt.Sprintf("%s: %s %s", e.Code, e.ResourceType, e.ResourceID)
    }
    return e.Code
}

func (e *ProviderError) Unwrap() error {
    return e.Err
}

func (e *ProviderError) Is(target error) bool {
    t, ok := target.(*ProviderError)
    if !ok {
        return false
    }
    return e.Code == t.Code
}
```

Usage pattern:

```go
// Returning a not-found error with metadata
return model.VM{}, &providererr.ProviderError{
    Code:         "not_found",
    Message:      "vm not found",
    ResourceType: "vm",
    ResourceID:   vmID,
}

// Checking on the caller side
if errors.Is(err, providererr.ErrNotFound) { ... }
```

Helper constructors like `NotFound(resourceType, id string) error` can be added for ergonomics, but the core type is what matters.

### Conformance Test Pattern (`providertest/statetest/statetest.go`)

Each conformance suite exports a single entry-point function:

```go
package statetest

import (
    "testing"
    "github.com/flames-hq/flames/provider/state"
)

// Run executes the full StateStore conformance suite against the given implementation.
func Run(t *testing.T, newStore func() state.StateStore) {
    t.Run("CreateAndGet", func(t *testing.T) { testCreateAndGet(t, newStore()) })
    t.Run("UpdateDesiredState", func(t *testing.T) { testUpdateDesiredState(t, newStore()) })
    t.Run("AssignVM", func(t *testing.T) { testAssignVM(t, newStore()) })
    t.Run("ConcurrentAssign", func(t *testing.T) { testConcurrentAssign(t, newStore()) })
    t.Run("ListPendingVMs", func(t *testing.T) { testListPendingVMs(t, newStore()) })
    t.Run("Events", func(t *testing.T) { testEvents(t, newStore()) })
    t.Run("Controllers", func(t *testing.T) { testControllers(t, newStore()) })
    t.Run("NotFoundErrors", func(t *testing.T) { testNotFoundErrors(t, newStore()) })
    // ...
}
```

The factory function `func() state.StateStore` lets each adapter pass its own constructor. The in-memory default's test file is trivial:

```go
// provider/state/memstate/memstate_test.go
package memstate_test

import (
    "testing"
    "github.com/flames-hq/flames/providertest/statetest"
    "github.com/flames-hq/flames/provider/state/memstate"
)

func TestConformance(t *testing.T) {
    statetest.Run(t, func() state.StateStore { return memstate.New() })
}
```

A future `provider/state/postgres/` adapter runs the exact same suite with a different factory.

### In-Memory Implementation Notes

**memstate** — `sync.RWMutex` protects a `map[string]model.VM`, `map[string]model.Controller`, and `[]model.Event`. `CreateVM` generates IDs with `crypto/rand` (UUIDv4-style hex, no external dependency). `AssignVM` holds a write lock and checks `ControllerID == ""` atomically.

**memblob** — `sync.RWMutex` over `map[string]storedBlob` where `storedBlob` holds `[]byte` data and `model.BlobMeta`. `Put` computes SHA-256 checksum and stores size. `Get` returns `io.NopCloser(bytes.NewReader(data))`.

**memcache** — `sync.RWMutex` over `map[string]cacheEntry` where `cacheEntry` holds `value []byte` and `expiresAt time.Time`. `Get` checks `time.Now().After(expiresAt)` and returns `ErrCacheMiss` if expired. No background reaper — expired entries are lazily evicted on access to keep the implementation simple. If the map grows unbounded in long-running tests, a periodic sweep can be added later.

**memqueue** — `sync.Mutex` over per-topic `[]queueEntry` slices. `Enqueue` appends; `Dequeue` scans for the first entry where `leaseExpiresAt.IsZero() || time.Now().After(leaseExpiresAt)`, sets `leaseExpiresAt = time.Now().Add(leaseTimeout)`, and returns it. `Ack` removes the entry. `Nack` resets `leaseExpiresAt` to `retryAt`. No background goroutines — lease expiry is checked at dequeue time.

**noop (ingress)** — All methods return `nil` error and zero-value results. `GetEndpoint` for an unknown VM returns `providererr.ErrNotFound`. Internally tracks registered VMs in a `map[string]model.Endpoint` so `GetEndpoint` can distinguish registered from unregistered, but performs no actual network operations.

## Dependencies

**External dependencies: none.** All default implementations use only the Go standard library (FR-002/NFR-002):

- `sync` — mutexes for concurrency safety
- `crypto/rand` — ID generation
- `time` — TTL, lease expiry
- `bytes`, `io` — blob streaming
- `crypto/sha256` — blob checksum
- `encoding/hex` — ID and checksum formatting
- `fmt`, `errors` — error formatting and wrapping

**Internal dependencies:**
- Every interface package imports `model` and `provider/providererr`
- Every default implementation imports its parent interface package
- Every conformance test imports its target interface package and `provider/providererr`

## Risks & Mitigations

**Interface bloat** — adding too many methods upfront locks in signatures that real adapters may struggle with. Impact is high since changing interfaces later is a breaking change for all adapters. We'll start with the minimum operation set from requirements and resist adding convenience methods to interfaces — those go in composing packages instead.

**Model types grow complex** — `VMSpec` sub-structs will accumulate fields as the project evolves, making the `CreateVM` signature implicitly wider. Medium impact since adapter authors must handle all fields. Mitigated by keeping `VMSpec` opaque to `StateStore` — it stores and returns it, not interprets it. Adapters serialize it as JSON/blob. Only `DesiredState`, `ObservedState`, and `ControllerID` are first-class indexed fields.

**In-memory defaults hide concurrency bugs** — mutex-protected maps serialize everything, masking race conditions that appear under real database contention. Medium impact since bugs surface only when switching to Postgres/SQLite. Mitigated by conformance tests that include explicit concurrent-access tests (AC-005), running all tests with `-race`, and documenting that adapters should also run conformance tests under concurrent load.

**Lazy cache eviction leaks memory** — no background reaper means expired entries stay in the map until accessed. Low impact, only affects long-running processes using memcache. Acceptable for development/testing since the in-memory cache is not a production backend. We can add an optional `Purge()` or periodic sweep later if needed.

**`Dequeue` non-blocking vs blocking** — callers must poll, adding latency. Low impact since this is a design trade-off, not a bug. Non-blocking is simpler to implement correctly across all backends, and callers (reconciler, scheduler) already run on tick-based loops. A `Subscribe` method can be added as a separate optional interface later if needed.
