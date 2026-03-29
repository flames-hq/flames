---
schema_version: 1
id: xF81mgjIio
created_at: 2026-03-29T15:59:14Z
updated_at: 2026-03-29T15:59:14Z
labels:
- DOCS
position: 3.0
---

# Controller Runtime Plan

The controller will be the host-side workhorse — it takes assignments from the control plane and turns them into running Firecracker VMs.

## What We Need to Build

A start flow that: polls for assigned work, allocates a private IP and tap device, sets up per-VM firewall rules, prepares the jailer directory, fetches kernel/rootfs artifacts, materializes the runtime rootfs (copy-on-write via OverlayFS for ephemeral mode), configures MMDS and vsock, launches Firecracker through Jailer, waits for guest agent auth/readiness, and reports observed state back.

## Cleanup

When a VM is done, we need to tear down everything: Firecracker process, jailer resources, tap device, firewall rules, temp disks, per-VM writable rootfs layers, stale sockets, jail dirs, and ingress registration. Cleanup needs to be idempotent — safe to run multiple times.

## Failure Handling

If a start fails mid-flight, we'll mark observed state as `failed`, append a detailed event, and run full cleanup. If the controller itself restarts, it needs to reconcile existing VM processes, rediscover leased work, and clean orphaned host resources.

## Rules We'll Follow

- One jailed root per `vm_id` with deterministic paths
- No shared writable runtime state between VMs
- Never mutate the shared base rootfs
- Duplicate start requests for the same `vm_id` converge, not duplicate
- Stop/delete on an already-stopped VM returns success
