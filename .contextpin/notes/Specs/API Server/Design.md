---
schema_version: 1
id: OHo6fbeUNN
created_at: 2026-03-31T00:31:13Z
updated_at: 2026-03-31T00:35:36Z
labels:
- SPEC-002
---
# Design

## Approach

Split the API into two layers: a `Service` struct that owns all business logic and speaks only in domain types (`model.*`, `providererr.*`), and thin transport adapters that translate between wire protocols and service calls. The service accepts provider interfaces via constructor injection (same pattern as SPEC-001's `memstate.New()`). The initial transport is an HTTP adapter using `net/http`; a future gRPC adapter would import the same `Service` and add its own encoding layer.

This was chosen over a router-handler-first design (where business logic lives inside HTTP handlers) because it makes the core logic testable without `httptest`, keeps transport concerns (idempotency keys, query param parsing, status codes) out of domain logic, and makes adding gRPC a matter of writing a new adapter package — not refactoring the service (FR-001, US-8, AC-013).

## Architecture

### Package Layout

```
github.com/flames-hq/flames/
├── (existing packages: model/, provider/, providertest/)
│
├── api/
│   ├── service.go              # Service struct + all business methods (FR-001–012)
│   └── service_test.go         # Tests call service methods directly with in-memory providers
│
├── transport/
│   └── httpapi/
│       ├── handler.go          # HTTP handler that wraps Service (FR-013–020)
│       ├── handler_test.go     # httptest-based integration tests
│       ├── errors.go           # providererr → HTTP status code + JSON body mapping
│       └── idempotency.go      # Idempotency-Key middleware (FR-017)
│
└── cmd/
    └── flames-api/
        └── main.go             # Wires providers + service + transport, starts server (FR-021–022)
```

### Dependency Graph

```
model  ←──  provider/providererr
  ↑              ↑
  │              │
  ├── provider/state, blob, cache, queue, ingress   (existing, unchanged)
  │              ↑
  │              │
  ├── api/       ──────────────────────────────────  imports provider interfaces + model + providererr
  │              ↑
  │              │
  └── transport/httpapi/  ─────────────────────────  imports api/ + model + providererr
                 ↑
                 │
     cmd/flames-api/  ────────────────────────────   imports everything, wires and starts
```

The `api/` package depends only on provider interfaces, `model`, and `providererr`. It has no knowledge of HTTP, JSON encoding, or any transport. The `transport/httpapi/` package imports `api/` and handles all HTTP-specific concerns. A future `transport/grpcapi/` would import `api/` identically.

### How It Fits the Existing Architecture

SPEC-001 established the pattern: interfaces in `provider/*/`, implementations in sub-packages, consumers accept interfaces via constructors. The `api.Service` is the first consumer of these interfaces, following the exact wiring pattern shown in SPEC-001's Design:

```go
// cmd/flames-api/main.go
ss := memstate.New()
wq := memqueue.New()
svc := api.New(ss, wq)
handler := httpapi.NewHandler(svc)
```

### Why `api/` is a Package, Not an Interface

The service layer is a concrete struct, not an interface. Transports call its methods directly — there is no `Transport` interface or `ServiceInterface` that transports implement or consume. This is intentional:

- There will only ever be one service implementation (the business logic is the business logic).
- An interface would add indirection with no polymorphism benefit.
- Transports are differentiated by wire format, not by behavior. Each transport is just a different way to call the same `Service` methods.
- Testing transports uses the real `Service` with in-memory providers — no mocking needed.

If a need for a service interface arises later (e.g., middleware chaining), it can be extracted from the concrete type at that point.

## Data Model

No new domain types are introduced. The service operates entirely on existing `model` types from SPEC-001:

| Service Method       | Input Types                                 | Output Types         |
|----------------------|---------------------------------------------|----------------------|
| `CreateVM`           | `model.VMSpec`                              | `string` (VM ID)     |
| `GetVM`              | `string` (VM ID)                            | `model.VM`           |
| `StopVM`             | `string` (VM ID)                            | —                    |
| `DeleteVM`           | `string` (VM ID)                            | —                    |
| `RegisterController` | `model.Controller`                          | —                    |
| `Heartbeat`          | `string` (controller ID), `model.Heartbeat` | —                    |
| `ListControllers`    | `model.ControllerFilter`                    | `[]model.Controller` |
| `ListEvents`         | `model.EventFilter`                         | `[]model.Event`      |

### Idempotency State

The idempotency store is a transport concern (only HTTP needs `Idempotency-Key` headers; gRPC has its own patterns). It lives in `transport/httpapi/` and uses a simple in-memory map:

```go
type idempotencyEntry struct {
    RequestHash [32]byte   // SHA-256 of request body
    StatusCode  int
    Body        []byte     // cached response body
    CreatedAt   time.Time
}
```

Keyed by the `Idempotency-Key` header value. Entries are evicted after 24 hours (lazy check on access). Protected by `sync.RWMutex` — same concurrency pattern as the in-memory providers.

## API / Interfaces

### Service (`api/service.go`)

```go
package api

import (
    "context"
    "github.com/flames-hq/flames/model"
    "github.com/flames-hq/flames/provider/state"
    "github.com/flames-hq/flames/provider/queue"
)

type Service struct {
    state state.StateStore
    queue queue.WorkQueue
}

func New(state state.StateStore, queue queue.WorkQueue) *Service {
    return &Service{state: state, queue: queue}
}

// VM operations

func (s *Service) CreateVM(ctx context.Context, spec model.VMSpec) (string, error)
func (s *Service) GetVM(ctx context.Context, vmID string) (model.VM, error)
func (s *Service) StopVM(ctx context.Context, vmID string) error
func (s *Service) DeleteVM(ctx context.Context, vmID string) error

// Controller operations

func (s *Service) RegisterController(ctx context.Context, c model.Controller) error
func (s *Service) Heartbeat(ctx context.Context, controllerID string, hb model.Heartbeat) error
func (s *Service) ListControllers(ctx context.Context, filter model.ControllerFilter) ([]model.Controller, error)

// Event operations

func (s *Service) ListEvents(ctx context.Context, filter model.EventFilter) ([]model.Event, error)
```

**Method behavior:**

- `CreateVM` calls `s.state.CreateVM`, then `s.state.AppendEvent` with type `vm.created` (FR-007). Returns the VM ID.
- `GetVM` delegates directly to `s.state.GetVM`. Returns the full `model.VM`.
- `StopVM` calls `s.state.UpdateVMDesiredState(ctx, vmID, model.DesiredStopped)`, then appends a `vm.stop_requested` event (FR-007).
- `DeleteVM` calls `s.state.UpdateVMDesiredState(ctx, vmID, model.DesiredDeleted)`, then appends a `vm.delete_requested` event (FR-007).
- `RegisterController` delegates to `s.state.RegisterController`.
- `Heartbeat` delegates to `s.state.UpdateControllerHeartbeat`. Returns `providererr.ErrNotFound` if the controller doesn't exist (FR-006, FR-010).
- `ListControllers` delegates to `s.state.ListControllers`, enforcing default limit of 100 and max of 1000 (same as `ListEvents`).
- `ListEvents` delegates to `s.state.ListEvents`, enforcing the default limit of 100 and max of 1000 before passing the filter through.

**New model type required (`model/controller.go`):**

```go
type ControllerFilter struct {
    Status string    `json:"status"`     // filter by status ("active", "draining", "offline")
    Limit  int       `json:"limit"`
}
```

Follows the same zero-value-means-ignore convention as `EventFilter`.

**StateStore addition required:**

The existing `StateStore` interface has `RegisterController` and `UpdateControllerHeartbeat` but no `ListControllers`. To satisfy FR-010, we need to add:

```go
ListControllers(ctx context.Context, filter model.ControllerFilter) ([]model.Controller, error)
```

This is a small addition to the existing interface and its implementations (`memstate`, conformance tests). It follows the same pattern as `ListEvents`.

### HTTP Transport (`transport/httpapi/handler.go`)

```go
package httpapi

import (
    "net/http"
    "github.com/flames-hq/flames/api"
)

type Handler struct {
    svc *api.Service
    mux *http.ServeMux
    idem *idempotencyStore
}

func NewHandler(svc *api.Service) *Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

`NewHandler` creates the handler and registers all routes on an internal `http.ServeMux`. The `Handler` implements `http.Handler` so it can be passed directly to `http.Server`.

**Route registration** (using Go 1.22+ `ServeMux` pattern matching):

```go
mux.HandleFunc("POST /v1/vms", h.createVM)
mux.HandleFunc("GET /v1/vms/{vm_id}", h.getVM)
mux.HandleFunc("POST /v1/vms/{vm_id}/stop", h.stopVM)
mux.HandleFunc("DELETE /v1/vms/{vm_id}", h.deleteVM)
mux.HandleFunc("POST /v1/controllers", h.registerController)
mux.HandleFunc("POST /v1/controllers/{controller_id}/heartbeat", h.heartbeat)
mux.HandleFunc("GET /v1/controllers", h.listControllers)
mux.HandleFunc("GET /v1/events", h.listEvents)
mux.HandleFunc("GET /healthz", h.healthz)
```

Each handler method is ~15-25 lines: decode request, call `h.svc.Method()`, encode response or error.

### Error Mapping (`transport/httpapi/errors.go`)

```go
func errorToStatus(err error) int {
    switch {
    case errors.Is(err, providererr.ErrNotFound):      return http.StatusNotFound           // 404
    case errors.Is(err, providererr.ErrAlreadyExists):  return http.StatusConflict           // 409
    case errors.Is(err, providererr.ErrConflict):       return http.StatusConflict           // 409
    default:                                            return http.StatusInternalServerError // 500
    }
}
```

The `writeError` helper extracts `*providererr.ProviderError` fields (code, message, resource_type, resource_id) via `errors.As` and writes the JSON error body (FR-016).

### Idempotency (`transport/httpapi/idempotency.go`)

Wraps mutation handlers. On each request with an `Idempotency-Key` header:

1. Compute SHA-256 of the request body.
2. Check the in-memory map. If the key exists and hashes match → replay the cached response (same status code + body).
3. If the key exists but hashes differ → return `409 Conflict` (FR-017).
4. If the key is new → execute the handler, cache the response, return it.

Uses `http.ResponseWriter` wrapping to capture the response before it's sent to the client.

### Request Logging Middleware

A `logMiddleware` wraps `Handler.ServeHTTP` and logs method, path, status code, and duration as JSON to stdout (NFR-004):

```json
{"method":"POST","path":"/v1/vms","status":202,"duration_ms":1.2}
```

### Entry Point (`cmd/flames-api/main.go`)

```go
func main() {
    addr := flag.String("addr", ":8080", "listen address")
    flag.Parse()

    ss := memstate.New()
    wq := memqueue.New()
    svc := api.New(ss, wq)
    handler := httpapi.NewHandler(svc)

    srv := &http.Server{Addr: *addr, Handler: handler}

    // Graceful shutdown on SIGINT/SIGTERM (NFR-005)
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(ctx)
    }()

    log.Printf("listening on %s", *addr)
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
```

## Dependencies

**External dependencies: none.** Consistent with NFR-002 and the project's zero-dependency constraint.

**Standard library packages used (new to this spec):**

- `net/http` — HTTP server, `ServeMux`, `ResponseWriter`
- `encoding/json` — request/response encoding
- `crypto/sha256` — idempotency key body hashing
- `flag` — CLI flag parsing for `--addr`
- `os/signal`, `syscall` — graceful shutdown
- `log` — request logging to stdout

**Internal dependencies:**

- `api/` imports: `model`, `provider/state`, `provider/queue`, `provider/providererr`
- `transport/httpapi/` imports: `api`, `model`, `provider/providererr`
- `cmd/flames-api/` imports: `api`, `transport/httpapi`, `provider/state/memstate`, `provider/queue/memqueue`

**Note on BlobStore and IngressProvider:** The service constructor takes only `StateStore` and `WorkQueue` for now. The current requirements (FR-001–012) don't exercise `BlobStore` (no image upload endpoints) or `IngressProvider` (no routing management). These can be added to the constructor when future specs require them (e.g., image management, VM networking). This keeps the initial surface minimal.

## Risks & Mitigations

**StateStore interface change** — Adding `ListControllers` modifies an existing interface from SPEC-001, which means updating `memstate`, the conformance suite, and any future adapters. Low risk since we're pre-v1 with no external consumers, but worth noting. Mitigated by keeping the addition minimal (one method, same pattern as `ListPendingVMs`) and updating the conformance tests in the same changeset.

**Idempotency store is in-memory** — Keys are lost on restart, so a retried request after server restart will be treated as new. This aligns with the provider tier model: in-memory for tests, SQLite for local dev, Postgres (or other databases) for prod. The idempotency store follows the same progression — in-memory is correct for this tier since all state is ephemeral anyway. When SQLite/Postgres StateStore adapters land, the idempotency store should be backed by the same database. For now, the 24-hour TTL eviction prevents unbounded memory growth.

**Service methods are thin wrappers** — Several service methods (e.g., `GetVM`, `ListControllers`) are direct pass-throughs to `StateStore`. This might feel like unnecessary indirection, but the service layer is where validation, authorization, event emission, and cross-provider orchestration will accumulate. Starting with thin wrappers establishes the pattern so those concerns have a home when they arrive.
