---
schema_version: 1
id: Ame9sxhfrm
created_at: 2026-03-29T15:59:13Z
updated_at: 2026-03-29T15:59:13Z
labels:
- DOCS
position: 1.0
---

# Provider Interfaces Plan

We'll put every piece of infrastructure behind a small, behavior-focused interface. This will let us swap backends (SQLite to Postgres, local disk to S3) without touching business logic.

## What We Need to Build

**StateStore** — where VMs, images, controllers, and events will live. Needs to handle desired/observed state with reconciliation-safe, atomic, concurrent-safe updates.

**BlobStore** — will store opaque objects with metadata: artifacts, snapshots, backups, exports. No POSIX filesystem semantics — just put, get, list by prefix.

**CacheStore** — ephemeral cache for hot metadata. Core logic needs to work fine without it. Will never be the only copy of anything.

**WorkQueue** — will drive async reconciliation and background jobs. At-least-once delivery, idempotent handlers. Enqueue by topic, dequeue with lease timeout, ack or nack.

**IngressProvider** — optional public exposure for VM services. Register/unregister endpoints by `vm_id`. OSS default will be a no-op.

## Day One Plan

Start with SQLite, local filesystem, in-memory cache, in-process queue, and no-op ingress. Zero external dependencies for a single-machine deployment. Hosted adapters (Postgres, S3, Redis, platform ingress) will come later.
