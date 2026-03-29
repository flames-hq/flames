---
schema_version: 1
id: 0wC56YMgr8
created_at: 2026-03-29T15:30:32Z
updated_at: 2026-03-29T15:46:05Z
labels:
- SPEC-001
position: 0.0
---
# Requirements

## Overview

Define the five core infrastructure provider interfaces (`StateStore`, `BlobStore`, `CacheStore`, `WorkQueue`, `IngressProvider`) as Go interfaces with behavior-focused contracts, structured error types, and `context.Context` on every method. Each interface ships with a no-op or in-memory default implementation so the control plane can start and pass conformance tests without any external dependencies.

## Background & Motivation

The Flames architecture mandates that core orchestration logic never depends on a specific database, object store, cache, queue, or ingress platform. All infrastructure access flows through narrow, backend-agnostic interfaces. This is a prerequisite for every subsequent milestone — the API server, scheduler, reconciler, and controller runtime all consume these interfaces.

The project is greenfield (no Go code exists yet). This spec covers the initial Go module setup, interface definitions, error types, domain value types used in interface signatures, default implementations, and conformance test suites.

## User Stories

- US-1: As a control-plane developer, I want a `StateStore` interface so that I can read and write VM, controller, and event records without coupling to SQLite or Postgres.
- US-2: As a control-plane developer, I want a `BlobStore` interface so that I can store and retrieve opaque artifacts without coupling to the local filesystem or S3.
- US-3: As a control-plane developer, I want a `CacheStore` interface so that I can use ephemeral caching without requiring Redis in development.
- US-4: As a control-plane developer, I want a `WorkQueue` interface so that I can enqueue and consume background jobs without choosing a queue backend upfront.
- US-5: As a control-plane developer, I want an `IngressProvider` interface so that VM service exposure is pluggable, with a safe no-op default.
- US-6: As a developer running the control plane locally, I want in-memory/no-op default implementations of all five providers so that `go run` works out of the box with zero external dependencies.
- US-7: As a contributor adding a new provider backend (e.g., Postgres `StateStore`), I want a shared conformance test suite per interface so that I can verify my implementation satisfies the contract.

## Functional Requirements

### Go Module & Package Layout

- FR-001: The project MUST have a `go.mod` at the repository root with module path `github.com/flames-hq/flames` targeting Go 1.24+.
- FR-002: Provider interfaces MUST live under a `provider/` top-level package, with one sub-package per provider type: `provider/state`, `provider/blob`, `provider/cache`, `provider/queue`, `provider/ingress`.
- FR-003: Default implementations MUST live in sibling packages: `provider/state/memstate`, `provider/blob/memblob`, `provider/cache/memcache`, `provider/queue/memqueue`, `provider/ingress/noop`.
- FR-004: Shared domain types referenced by interface signatures (e.g., `VM`, `Controller`, `Event`, `BlobMeta`, `Job`, `Endpoint`, state enums) MUST live in a `model/` top-level package, not inside any provider package.
- FR-005: Structured error types MUST live in a `provider/providererrors` package (or similar) importable by all providers without circular dependencies.

### StateStore Interface

- FR-010: `StateStore` MUST expose at minimum the following methods, each accepting `context.Context` as the first parameter:
  - `CreateVM(ctx, spec) -> (vm_id, error)`
  - `GetVM(ctx, vm_id) -> (vm, error)`
  - `UpdateVMDesiredState(ctx, vm_id, desired_state) -> error`
  - `UpdateVMObservedState(ctx, vm_id, observed_state, controller_id) -> error`
  - `AssignVM(ctx, vm_id, controller_id) -> error`
  - `ListPendingVMs(ctx) -> ([]vm, error)`
  - `AppendEvent(ctx, event) -> error`
  - `ListEvents(ctx, filter) -> ([]event, error)`
  - `RegisterController(ctx, controller) -> error`
  - `UpdateControllerHeartbeat(ctx, controller_id, heartbeat) -> error`
- FR-011: `StateStore` implementations MUST guarantee atomic updates — a call to `AssignVM` must not leave the record in a partially updated state.
- FR-012: `StateStore` implementations MUST be safe for concurrent use by multiple goroutines.
- FR-013: The default in-memory `StateStore` implementation (`memstate`) MUST use mutexes or equivalent to satisfy FR-012 and store all data in Go maps/slices.

### BlobStore Interface

- FR-020: `BlobStore` MUST expose at minimum the following methods, each accepting `context.Context` as the first parameter:
  - `Put(ctx, key, reader, metadata) -> error`
  - `Get(ctx, key) -> (reader, error)`
  - `Head(ctx, key) -> (blob_meta, error)`
  - `Delete(ctx, key) -> error`
  - `Exists(ctx, key) -> (bool, error)`
  - `List(ctx, prefix) -> ([]blob_meta, error)`
- FR-021: The blob metadata model MUST include: `Key`, `Size`, `Checksum`, `ContentType`, and a `Metadata` map for arbitrary key-value pairs.
- FR-022: `BlobStore` MUST NOT require POSIX directory semantics — keys are flat strings with optional prefix-based listing.
- FR-023: The default in-memory `BlobStore` implementation (`memblob`) MUST store blob contents in byte slices in memory.

### CacheStore Interface

- FR-030: `CacheStore` MUST expose at minimum the following methods, each accepting `context.Context` as the first parameter:
  - `Get(ctx, key) -> ([]byte, error)` — returns `ErrCacheMiss` on cache miss
  - `Set(ctx, key, value, ttl) -> error`
  - `Delete(ctx, key) -> error`
- FR-031: `CacheStore` MAY expose `CompareAndSet(ctx, key, old_value, new_value, ttl) -> (bool, error)` as an optional method (separate interface or method that returns `ErrNotSupported`).
- FR-032: The default in-memory `CacheStore` implementation (`memcache`) MUST respect TTL — expired entries MUST NOT be returned by `Get`.
- FR-033: Core control-plane logic MUST NOT depend on `CacheStore` for correctness — the system must function identically (slower, but correctly) if every cache call returns a miss.

### WorkQueue Interface

- FR-040: `WorkQueue` MUST expose at minimum the following methods, each accepting `context.Context` as the first parameter:
  - `Enqueue(ctx, topic, payload) -> (job_id, error)`
  - `Dequeue(ctx, topic, lease_timeout) -> (job, error)` — blocks or returns `ErrNoJobs` if empty
  - `Ack(ctx, job_id) -> error`
  - `Nack(ctx, job_id, retry_at) -> error`
- FR-041: `WorkQueue` MUST provide at-least-once delivery semantics — a job that is dequeued but not acknowledged before the lease expires MUST become available for redelivery.
- FR-042: The default in-memory `WorkQueue` implementation (`memqueue`) MUST implement lease tracking and automatic redelivery of unacknowledged jobs after lease expiry.

### IngressProvider Interface

- FR-050: `IngressProvider` MUST expose at minimum the following methods, each accepting `context.Context` as the first parameter:
  - `RegisterEndpoint(ctx, vm_id, route, target) -> (endpoint, error)`
  - `UnregisterEndpoint(ctx, vm_id) -> error`
  - `GetEndpoint(ctx, vm_id) -> (endpoint, error)`
- FR-051: The default no-op `IngressProvider` implementation MUST accept all calls without error and return empty/zero-value results (except `GetEndpoint` which MUST return `ErrNotFound` for unknown VMs).

### Structured Error Types

- FR-060: Provider errors MUST be structured types (not bare `errors.New` strings) that support `errors.Is` / `errors.As` unwrapping.
- FR-061: The following error sentinels/types MUST be defined at minimum:
  - `ErrNotFound` — resource does not exist
  - `ErrAlreadyExists` — resource already exists (conflict on create)
  - `ErrConflict` — state conflict (e.g., stale update, concurrent modification)
  - `ErrNotSupported` — operation not supported by this implementation
  - `ErrCacheMiss` — cache key not found (CacheStore-specific)
  - `ErrNoJobs` — queue is empty (WorkQueue-specific)
- FR-062: Error types SHOULD carry structured metadata where applicable (e.g., `ErrNotFound` should include the resource type and ID that was not found).

### Conformance Tests

- FR-070: Each provider interface MUST have a conformance test suite written as a Go test helper function that accepts an interface instance and runs a standard battery of behavioral tests against it.
- FR-071: Conformance tests MUST be runnable via `go test ./provider/...` and MUST pass for all default implementations.
- FR-072: Conformance test suites MUST live in a `providertest/` package (or `provider/<type>/<type>test/`) and be importable by future adapter packages (e.g., a Postgres `StateStore` adapter runs the same conformance suite).

## Non-Functional Requirements

- NFR-001: **Portability** — All code (interfaces, default implementations, conformance tests) MUST compile and pass tests on both macOS (darwin/amd64, darwin/arm64) and Linux (linux/amd64) with no CGo requirement for default implementations.
- NFR-002: **Zero external dependencies for defaults** — Default implementations MUST NOT import any third-party modules. Only the Go standard library is allowed.
- NFR-003: **Performance (in-memory defaults)** — In-memory `StateStore` and `CacheStore` operations MUST complete in under 1ms for datasets up to 10,000 records (this is a sanity baseline, not a production target).
- NFR-004: **Concurrency safety** — All default implementations MUST be safe for concurrent use and MUST pass `go test -race` without data-race warnings.
- NFR-005: **API surface minimality** — Interfaces MUST expose only the minimum operations specified in this requirements doc. Additional convenience methods belong in higher-level packages that compose these interfaces.

## Acceptance Criteria

- AC-001: Given a freshly cloned repository, when `go build ./...` is run, then it completes with zero errors.
- AC-002: Given a freshly cloned repository, when `go test ./...` is run, then all tests pass, including conformance suites for all five default implementations.
- AC-003: Given a freshly cloned repository, when `go test -race ./...` is run, then no data-race warnings are reported.
- AC-004: Given the in-memory `StateStore`, when `CreateVM` is called followed by `GetVM` with the returned ID, then the same VM spec is returned.
- AC-005: Given the in-memory `StateStore`, when `AssignVM` is called concurrently from two goroutines for the same VM, then exactly one assignment succeeds and the store is not corrupted.
- AC-006: Given the in-memory `BlobStore`, when `Put` is called with a key and data, then `Get` returns the same data, `Head` returns correct metadata, and `Exists` returns true.
- AC-007: Given the in-memory `CacheStore`, when `Set` is called with a 1-second TTL, then `Get` returns the value immediately but returns a miss after the TTL expires.
- AC-008: Given the in-memory `WorkQueue`, when a job is enqueued and dequeued but not acknowledged before the lease expires, then the job becomes available for dequeue again.
- AC-009: Given the no-op `IngressProvider`, when `RegisterEndpoint` is called, then no error is returned; when `GetEndpoint` is called for an unregistered VM, then `ErrNotFound` is returned.
- AC-010: Given any default provider implementation, when the conformance test suite is run against it, then all conformance tests pass.
- AC-011: Given the `providertest` conformance test helpers, when a new adapter implementation is written and passed to the helper, then the same tests execute against the new adapter without modification.

## Constraints & Context from Architecture Docs

The following architectural decisions constrain this spec:

- **VM state model**: Desired states are `running`, `stopped`, `deleted`. Observed states are `pending`, `scheduled`, `preparing`, `starting`, `running`, `stopping`, `stopped`, `failed`, `deleted`. The `StateStore` interface signatures must use these enums.
- **Reconciliation model**: The control plane stores desired state and async workers drive convergence. `StateStore` is the source of truth — not caches, not queues.
- **OSS defaults**: SQLite (`StateStore`), local filesystem (`BlobStore`), in-memory (`CacheStore`), in-process (`WorkQueue`), no-op (`IngressProvider`). This spec covers the in-memory/no-op layer only; SQLite and local-filesystem adapters are separate work.
- **Idempotent handlers**: All `WorkQueue` consumers must be idempotent because the queue provides at-least-once delivery.
- **Security boundary**: OSS flames is unauthenticated and single-tenant. Provider interfaces do not need auth/authz hooks.
- **Testing strategy**: Provider conformance tests are a first-class test layer (`make test-provider`), expected to run on macOS and Linux.

## Open Questions

- [RESOLVED] Go module path — `github.com/flames-hq/flames`.
- [RESOLVED] `CacheStore.Get` returns `([]byte, error)` with `ErrCacheMiss` on miss — consistent with the `(value, error)` pattern used across all other provider methods.
