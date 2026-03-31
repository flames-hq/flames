---
schema_version: 1
id: UJFOfCm12A
created_at: 2026-03-31T00:45:31Z
updated_at: 2026-03-31T00:47:17Z
labels:
- SPEC-002
---
# Tasks

## Phase 1: Setup

- [x] TASK-001 [P] — Add `ControllerFilter` type to model and extend `StateStore` interface with `ListControllers(ctx, filter)` method, implement in `memstate`, add conformance tests in `statetest` (`model/controller.go`, `provider/state/state.go`, `provider/state/memstate/memstate.go`, `provider/state/memstate/memstate_test.go`, `providertest/statetest/statetest.go`)
- [x] TASK-002 [P] — Create package scaffolding for `api/`, `transport/httpapi/`, and `cmd/flames-api/` (`api/`, `transport/httpapi/`, `cmd/flames-api/`)

## Phase 2: Core Implementation

- [x] TASK-003 [depends: TASK-001] [US-1, US-2, US-3, US-4, US-5, US-6, US-8] — Implement `api.Service` struct with constructor injection (`New(state, queue)`) and all 8 business methods: `CreateVM`, `GetVM`, `StopVM`, `DeleteVM`, `RegisterController`, `Heartbeat`, `ListControllers`, `ListEvents` — including event emission for mutations and limit enforcement on list operations (`api/service.go`)
- [x] TASK-004 [depends: TASK-003] — Service layer tests: cover all acceptance criteria AC-001 through AC-007 using in-memory providers, no HTTP required (`api/service_test.go`)
- [x] TASK-005 [depends: TASK-002] [P] — Implement `providererr` → HTTP status code mapping (`errorToStatus`) and structured JSON error response writer (`writeError`) that extracts `ProviderError` fields via `errors.As` (`transport/httpapi/errors.go`)
- [x] TASK-006 [depends: TASK-002] [P] — Implement `Idempotency-Key` middleware: in-memory store with `sync.RWMutex`, SHA-256 body hashing, response capturing via `ResponseWriter` wrapper, 24-hour lazy TTL eviction, `409 Conflict` on key reuse with different body (`transport/httpapi/idempotency.go`)

## Phase 3: HTTP Transport

- [x] TASK-007 [depends: TASK-003, TASK-005, TASK-006] [US-1, US-2, US-3, US-4, US-5, US-6] — Implement `httpapi.Handler` with `ServeHTTP`, route registration on Go 1.22+ `ServeMux`, all 9 handler methods (createVM, getVM, stopVM, deleteVM, registerController, heartbeat, listControllers, listEvents, healthz), request logging middleware, and idempotency wrapping on mutation routes (`transport/httpapi/handler.go`)
- [x] TASK-008 [depends: TASK-007] — HTTP transport integration tests using `httptest.Server`: cover AC-008 through AC-012 including status codes, JSON error bodies, idempotency replay/conflict, healthz, and `go test -race` (`transport/httpapi/handler_test.go`)

## Phase 4: Wiring & Verification

- [x] TASK-009 [depends: TASK-007] [US-7] — Implement entry point: wire in-memory providers → service → HTTP handler, `--addr` flag with `:8080` default, graceful shutdown on SIGINT/SIGTERM with 10s drain timeout (`cmd/flames-api/main.go`)
- [x] TASK-010 [depends: TASK-008, TASK-009] — Full verification: `go build ./...`, `go test ./...`, `go test -race ./...`, manual smoke test via `go run ./cmd/flames-api` + curl against all endpoints, validate AC-011
