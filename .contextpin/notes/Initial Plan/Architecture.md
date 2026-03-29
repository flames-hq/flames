---
schema_version: 1
id: jMXdjyBKMO
created_at: 2026-03-29T15:59:13Z
updated_at: 2026-03-29T16:04:40Z
labels:
- DOCS
position: 0.0
---

# Architecture Plan

We're building Flames as an open-source control plane and orchestrator for Firecracker microVMs.

## What We're Going For

A split architecture — a stateless **control plane** that handles API, scheduling, image coordination, and ingress metadata, and **controllers** that run on EC2 with Firecracker + Jailer and own the actual VM processes, networking, and host-local execution.

## First Milestone

One region, one control plane, one controller — all on a single machine for the OSS default. We need VM create/stop/delete working, a settings-driven VM model (no separate "mode" field for jobs vs services), and a Dockerfile/OCI image pipeline that produces bootable Firecracker artifacts.

## Key Decisions We've Made

- **Go** for the OSS core
- **Reconciler-based** control plane — desired state in DB, async workers converge toward it
- **Required guest agent** in every VM from day one
- **Provider interfaces** for all infrastructure (state, blob, cache, queue, ingress) so we can swap backends later
- **OSS defaults**: SQLite + local filesystem + in-memory cache + in-process queue
- **Three repos**: `flames` (OSS core), `flames-platform` (hosted layer), `flames-ops` (infra ops)

## Build Order

1. Repo skeleton and ADRs
2. Local controller that boots one hardcoded Firecracker VM
3. Add Jailer and cleanup
4. Control-plane API with `POST /v1/vms`
5. Provider interfaces for StateStore, BlobStore, CacheStore, WorkQueue
6. SQLite persistence for VM state
7. Controller heartbeats and polling
8. OCI image ingestion and rootfs conversion
9. Guest agent and standardized bootstrap
10. Same-host network isolation and per-VM policy
11. Controller-local ingress abstraction
12. Artifact model shaped for future snapshot lineage
13. Local filesystem BlobStore and in-memory CacheStore
14. Multi-controller scheduling

## What We're Not Building Yet

No live migration, no multi-region HA, no persistent volumes, no custom domains, no in-place VM config mutation, no direct SSH as a product feature.
