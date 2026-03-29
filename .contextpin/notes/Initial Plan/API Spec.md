---
schema_version: 1
id: MTrEQIn3HG
created_at: 2026-03-29T15:59:13Z
updated_at: 2026-03-29T15:59:13Z
labels:
- DOCS
position: 0.0
---

# API Plan

We'll expose a JSON-over-HTTP API from the control plane. All mutations will be async — callers get back an acceptance, not a completion.

## What We'll Build First

A small surface: create a VM, read its state, stop it, delete it. Create and inspect images. List controllers and events. We're keeping it intentionally minimal.

## How VM Creation Will Work

Callers send an image reference plus configuration: resources (CPU/memory), runtime (command, env, ports), lifecycle (timeout, auto-delete, restart policy), network (ingress, egress), storage (rootfs mode, ephemeral disk), placement (pool, region), and metadata (labels). The API returns `202 Accepted` with a server-generated `vm_id` — progress gets tracked through state transitions and events.

## Idempotency

We'll require an `Idempotency-Key` header on every mutation. Same key + same body returns the original result. Same key + different body returns 409 conflict. This prevents duplicate VMs from retried requests.

## Auth Approach

No built-in app-layer auth in the OSS API. Operators will put it behind their own perimeter (VPN, reverse proxy, gateway). User auth will live in the platform layer later.

## What Comes Later

Interactive exec, live log streaming, snapshot APIs, file browser, and custom domains are all planned but won't be in the first API.
