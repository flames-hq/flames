---
schema_version: 1
id: KvKLya2umi
created_at: 2026-03-31T00:12:50Z
updated_at: 2026-03-31T00:12:50Z
labels:
  - SPEC-002
---

# Requirements

## Overview

The API Server is the control plane for Flames, providing VM lifecycle management, controller registration and heartbeats, and event querying. It is the first real consumer of the provider interfaces (StateStore, BlobStore, WorkQueue, IngressProvider). The API logic is transport-agnostic — a `Service` layer owns all business logic, and transports (HTTP, gRPC, etc.) are thin adapters that decode requests, call the service, and encode responses. The initial transport is HTTP; the architecture MUST allow adding new transports without modifying the service layer. Ships as a single `go run`-able binary with zero external dependencies when using in-memory defaults.

## User Stories

- US-1: As an operator, I want to create a VM via an API so that I can provision Firecracker microVMs without direct access to the control plane internals.
- US-2: As an operator, I want to inspect a VM's current desired and observed state so that I can monitor convergence and troubleshoot failures.
- US-3: As an operator, I want to stop or delete a VM via an API so that I can manage the full VM lifecycle through a single API surface.
- US-4: As a controller node, I want to register myself with the control plane so that the scheduler knows I exist and can assign VMs to me.
- US-5: As a controller node, I want to send periodic heartbeats with my capacity so that the control plane has an accurate view of cluster health.
- US-6: As an operator, I want to list and filter events so that I can audit what happened to a specific VM or across the cluster.
- US-7: As a developer, I want to start a fully functional API server with `go run ./cmd/flames-api` so that I can develop and test locally without external dependencies.
- US-8: As a platform team, I want to swap the transport layer (e.g., HTTP to gRPC) without rewriting business logic so that we can evolve the API surface independently of the domain.

## Functional Requirements

### Service Layer (Transport-Agnostic)

- FR-001: The system MUST implement a `Service` type that contains all API business logic as plain Go methods accepting and returning domain types — no HTTP, no gRPC, no transport concepts.
- FR-002: The service MUST provide a `CreateVM` operation that accepts a `VMSpec` and returns the created VM's ID.
- FR-003: The service MUST provide a `GetVM` operation that accepts a VM ID and returns the full VM resource including desired state, observed state, controller assignment, and timestamps.
- FR-004: The service MUST provide a `StopVM` operation that accepts a VM ID and sets the desired state to `stopped`.
- FR-005: The service MUST provide a `DeleteVM` operation that accepts a VM ID and sets the desired state to `deleted`.
- FR-006: The service MUST return typed errors (mapping to `providererr` types) that transports can translate to their native error representations (e.g., HTTP 404, gRPC NOT_FOUND).
- FR-007: The service MUST emit events (via `StateStore.AppendEvent`) for each VM lifecycle mutation: `vm.created`, `vm.stop_requested`, `vm.delete_requested`.
- FR-008: The service MUST provide a `RegisterController` operation that accepts a controller's ID, labels, and initial capacity.
- FR-009: The service MUST provide a `Heartbeat` operation that accepts a controller ID and heartbeat payload (status and capacity).
- FR-010: The service MUST provide a `ListControllers` operation that accepts filter parameters (status, limit) and returns matching controllers with their current status and last heartbeat time. Default limit: 100, max: 1000.
- FR-011: The service MUST provide a `ListEvents` operation that accepts filter parameters (VM ID, controller ID, type, since, limit) and returns events ordered by creation time (newest first).
- FR-012: The service MUST accept provider interfaces via constructor injection so that implementations can be swapped without changing service code.

### Transport: HTTP (Initial Implementation)

- FR-013: The HTTP transport MUST be a thin adapter that decodes HTTP requests into service calls and encodes service responses into HTTP responses.
- FR-014: The HTTP transport MUST expose the following routes mapping to service operations:
  - `POST /v1/vms` → `CreateVM` → `202 Accepted`
  - `GET /v1/vms/{vm_id}` → `GetVM` → `200 OK`
  - `POST /v1/vms/{vm_id}/stop` → `StopVM` → `202 Accepted`
  - `DELETE /v1/vms/{vm_id}` → `DeleteVM` → `202 Accepted`
  - `POST /v1/controllers` → `RegisterController` → `201 Created`
  - `POST /v1/controllers/{controller_id}/heartbeat` → `Heartbeat` → `200 OK`
  - `GET /v1/controllers` → `ListControllers` → `200 OK` (supports `status` and `limit` query params)
  - `GET /v1/events` → `ListEvents` → `200 OK`
- FR-015: The HTTP transport MUST accept and return `application/json` for all endpoints.
- FR-016: The HTTP transport MUST return structured error responses with `code`, `message`, and where applicable `resource_type` and `resource_id` fields, translating from service-layer errors.
- FR-017: The HTTP transport MUST support an `Idempotency-Key` header on mutation endpoints (`POST`, `DELETE`). The same key with the same body MUST return the original response; the same key with a different body MUST return `409 Conflict`.
- FR-018: All mutation endpoints MUST return `202 Accepted` (not `200 OK`) to reflect the async nature of VM operations — the desired state is recorded, but convergence happens asynchronously.
- FR-019: The HTTP transport MUST support event filtering via query parameters: `vm_id`, `controller_id`, `type`, `since` (RFC 3339 timestamp), and `limit` (default 100, max 1000).
- FR-020: The HTTP transport MUST expose `GET /healthz` that returns `200 OK` with `{"status": "ok"}` for liveness checks.

### Server Infrastructure

- FR-021: The system MUST provide a `cmd/flames-api/main.go` entry point that wires up the service with in-memory providers and the HTTP transport, runnable via `go run ./cmd/flames-api`.
- FR-022: The system MUST accept a `--addr` flag (default `:8080`) to configure the listen address.
- FR-023: The service layer MUST be testable independently of any transport — tests call service methods directly with in-memory providers, no HTTP server required.

## Non-Functional Requirements

- NFR-001: Performance — API responses for single-resource reads (`GET /v1/vms/{id}`) MUST complete in < 10ms when using in-memory providers (p99, measured without network latency).
- NFR-002: Stdlib-only — The service layer and HTTP transport MUST use only the Go standard library with zero external dependencies, consistent with the project's zero-dependency constraint. Future transports (e.g., gRPC) MAY introduce dependencies scoped to their own package.
- NFR-003: Concurrency — The server MUST be safe for concurrent requests. All provider access is already concurrency-safe; the HTTP layer MUST NOT introduce new race conditions (e.g., in idempotency key tracking).
- NFR-004: Observability — The server MUST log each request with method, path, status code, and duration to stdout in a structured format.
- NFR-005: Graceful shutdown — The server MUST handle SIGINT/SIGTERM and drain in-flight requests before exiting (timeout: 10 seconds).

## Acceptance Criteria

### Service Layer

- AC-001: Given in-memory providers, when I call `service.CreateVM` with a valid `VMSpec`, then I get back a VM ID and `service.GetVM` returns a VM with observed state `pending`.
- AC-002: Given a VM exists, when I call `service.StopVM`, then `service.GetVM` returns desired state `stopped`.
- AC-003: Given a VM exists, when I call `service.DeleteVM`, then `service.GetVM` returns desired state `deleted`.
- AC-004: Given no VM with that ID exists, when I call `service.GetVM`, then I receive a `not_found` typed error.
- AC-005: Given in-memory providers, when I call `service.RegisterController`, then `service.ListControllers` includes the new controller.
- AC-006: Given a registered controller, when I call `service.Heartbeat` with updated capacity, then `service.ListControllers` reflects the new capacity.
- AC-007: Given multiple events exist, when I call `service.ListEvents` with a VM ID filter and limit of 10, then I receive only matching events, newest first, at most 10.

### HTTP Transport

- AC-008: Given the HTTP server is running, when I `POST /v1/vms` with a valid `VMSpec`, then I receive `202 Accepted` with a JSON body containing `id`.
- AC-009: Given no VM with that ID exists, when I `GET /v1/vms/{nonexistent}`, then I receive `404` with `{"code":"not_found","message":"vm not found","resource_type":"vm","resource_id":"..."}`.
- AC-010: Given I send `POST /v1/vms` with `Idempotency-Key: abc123` and body A, when I resend the same request, then I receive the same response. When I send `Idempotency-Key: abc123` with body B, then I receive `409 Conflict`.
- AC-011: Given `go run ./cmd/flames-api` is executed with no flags, the server starts on `:8080` and `GET /healthz` returns `{"status":"ok"}`.
- AC-012: Given concurrent requests hit the server, when I run `go test -race` against both the service and HTTP transport packages, then no race conditions are detected.

### Transport Independence

- AC-013: Given the service layer, when I write a new transport adapter (mock or real), then I can expose all service operations without modifying the service package — only the new transport package and the wiring in `cmd/`.

## Open Questions

- None — the provider interfaces are well-defined, and the roadmap clearly scopes this as the HTTP layer over existing abstractions.
