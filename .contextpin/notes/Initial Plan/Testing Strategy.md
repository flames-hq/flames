---
schema_version: 1
id: UpGdvCM5gX
created_at: 2026-03-29T15:59:17Z
updated_at: 2026-03-29T15:59:17Z
labels:
- DOCS
---

# Testing Plan

We'll split testing across layers to match the dev workflow — most things will run on any machine, runtime tests will need Linux + KVM.

## What We'll Build

**Unit tests** — state transitions, scheduler logic, API validation, guest-agent message parsing. Will run anywhere including macOS.

**Provider conformance tests** — every provider implementation (SQLite, Postgres, local FS, S3, in-memory, Redis) will need to pass the same contract test suite. This ensures swapping backends doesn't break behavior.

**Integration tests** — API-to-state-store flow, scheduler assignment, idempotency-key behavior, bootstrap token lifecycle, ingress registration. Can run without real Firecracker if the controller is stubbed.

**Controller runtime tests** — will need Linux + KVM. Cover Firecracker boot, Jailer setup, MMDS bootstrap, vsock auth, tap networking, rootfs materialization, cleanup and orphan recovery.

**End-to-end smoke tests** — create a short-lived VM and verify it exits, create a long-running VM and verify readiness, stop and delete cleanly, retry with the same idempotency key and verify no duplicate.

## Release Gate

Before we ship anything, it needs to pass: unit tests, provider conformance tests, one Linux runtime boot test, and one end-to-end smoke test.
