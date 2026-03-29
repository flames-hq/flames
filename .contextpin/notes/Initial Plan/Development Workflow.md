---
schema_version: 1
id: QRsqBRTNVg
created_at: 2026-03-29T15:59:16Z
updated_at: 2026-03-29T15:59:16Z
labels:
- DOCS
position: 6.0
---

# Development Workflow Plan

Firecracker needs Linux + KVM, but we'll do everyday development on macOS. The workflow splits accordingly.

## How We'll Work

**macOS (daily driver)** — control-plane API, scheduler, reconciler, provider interfaces, SQLite, local BlobStore, guest-agent code, unit tests, contract tests that don't need KVM.

**Linux EC2 (on-demand)** — Firecracker boot tests, Jailer tests, tap networking, vsock, rootfs materialization, cleanup/reconciliation, controller integration tests.

## Keeping It Cheap

We'll develop on Mac full-time and only spin up a controller-capable EC2 instance when we need runtime tests. Point the local control plane at that controller (SSH tunnel, Tailscale, or WireGuard), run the tests, stop the instance. No always-on infra for dev.

## Test Commands We'll Set Up

- `make test-unit` — runs on macOS
- `make test-integration` — needs Linux + KVM
- `make test-controller-runtime` — needs Firecracker + Jailer on Linux

## CI Direction

Standard runners for unit tests on every change. Dedicated KVM-capable runners for runtime tests. At least one end-to-end smoke test before any release.
