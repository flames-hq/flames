---
schema_version: 1
id: Q5ExH3L2xo
created_at: 2026-03-29T16:16:26Z
updated_at: 2026-03-29T17:30:00Z
labels:
- SPEC-001
---
# Tasks

## Phase 1: Foundation

- [x] TASK-001 — Initialize Go module and create directory skeleton (`go.mod`, `model/`, `provider/providererr/`, `provider/state/memstate/`, `provider/blob/memblob/`, `provider/cache/memcache/`, `provider/queue/memqueue/`, `provider/ingress/noop/`, `providertest/statetest/`, `providertest/blobtest/`, `providertest/cachetest/`, `providertest/queuetest/`, `providertest/ingresstest/`)
- [x] TASK-002 [depends: TASK-001] [US-1, US-2, US-4, US-5] — Define shared domain model types: VM, VMSpec, DesiredState/ObservedState enums, Controller, Heartbeat, CapacityInfo, Event, EventFilter, BlobMeta, Job, Endpoint, Route, Target (`model/vm.go`, `model/controller.go`, `model/event.go`, `model/blob.go`, `model/job.go`, `model/endpoint.go`)
- [x] TASK-003 [depends: TASK-002] — Define structured error types with sentinel errors, ProviderError type, Is/Unwrap support, and resource metadata (`provider/providererr/errors.go`)

## Phase 2: Provider Implementation

- [x] TASK-004 [depends: TASK-003] [P] [US-1, US-6, US-7] — StateStore interface + memstate default (mutex-protected maps, atomic AssignVM, crypto/rand ID generation) + statetest conformance suite including concurrent-access tests (`provider/state/state.go`, `provider/state/memstate/memstate.go`, `providertest/statetest/statetest.go`, `provider/state/memstate/memstate_test.go`)
- [x] TASK-005 [depends: TASK-003] [P] [US-2, US-6, US-7] — BlobStore interface + memblob default (byte-slice storage, SHA-256 checksums, io.ReadCloser streaming) + blobtest conformance suite (`provider/blob/blob.go`, `provider/blob/memblob/memblob.go`, `providertest/blobtest/blobtest.go`, `provider/blob/memblob/memblob_test.go`)
- [x] TASK-006 [depends: TASK-003] [P] [US-3, US-6, US-7] — CacheStore interface + optional AtomicCacheStore interface + memcache default (lazy TTL expiry, both interfaces implemented) + cachetest conformance suite (`provider/cache/cache.go`, `provider/cache/memcache/memcache.go`, `providertest/cachetest/cachetest.go`, `provider/cache/memcache/memcache_test.go`)
- [x] TASK-007 [depends: TASK-003] [P] [US-4, US-6, US-7] — WorkQueue interface + memqueue default (per-topic slices, lease tracking, automatic redelivery on expiry) + queuetest conformance suite (`provider/queue/queue.go`, `provider/queue/memqueue/memqueue.go`, `providertest/queuetest/queuetest.go`, `provider/queue/memqueue/memqueue_test.go`)
- [x] TASK-008 [depends: TASK-003] [P] [US-5, US-6, US-7] — IngressProvider interface + noop default (accepts all calls, tracks registered endpoints, ErrNotFound for unknown VMs) + ingresstest conformance suite (`provider/ingress/ingress.go`, `provider/ingress/noop/noop.go`, `providertest/ingresstest/ingresstest.go`, `provider/ingress/noop/noop_test.go`)

## Phase 3: Verification

- [x] TASK-009 [depends: TASK-004, TASK-005, TASK-006, TASK-007, TASK-008] — Full build and test verification: `go build ./...`, `go test ./...`, `go test -race ./...`, validate all acceptance criteria AC-001 through AC-011
