---
schema_version: 1
id: mFeFq5i9Gr
created_at: 2026-03-29T15:59:15Z
updated_at: 2026-03-29T15:59:15Z
labels:
- DOCS
position: 4.0
---

# State Machine and DB Plan

We'll store desired state and observed state separately in the control plane. Controllers report what's actually happening. Reconcilers close the gap.

## State Flow We'll Implement

VMs will move through: `pending → scheduled → preparing → starting → running`. Stop: `running → stopping → stopped`. Delete: `stopped → deleted`. Failures can happen from `preparing`, `starting`, or `running`.

The API will set desired state (`running`, `stopped`, or `deleted`). Controllers will report observed state. Reconcilers will compare and issue work until they converge.

## What We Need to Persist

VMs (desired/observed state, image, controller assignment, spec, timestamps), bootstrap tokens (per-VM, hashed, with expiry), images (source, artifact manifest), controllers (status, labels, capacity, last heartbeat), lifecycle events (append-only), audit events (who did what), and idempotency keys.

## Concurrency Rules

Controller assignments need to be atomic. Heartbeat updates can't overwrite unrelated state. Idempotency keys prevent duplicate VM creation. Bootstrap token validation needs to safely reject stale or reused sessions.

## Events

Every VM create, stop, delete, and failure will emit an event. Every controller assignment and ingress change will emit an event. Audit events will additionally record who initiated the action.

## Storage Plan

We'll start with SQLite for the OSS single-instance default. Postgres comes later for hosted deployments that need higher concurrency. Snapshot lineage tables, file backup catalogs, and account/tenant tables are all deferred.
