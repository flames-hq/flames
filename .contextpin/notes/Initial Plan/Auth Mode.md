---
schema_version: 1
id: 6eafJ8XglR
created_at: 2026-03-29T15:59:17Z
updated_at: 2026-03-29T15:59:17Z
labels:
- DOCS
---

# Auth and Security Plan

We're keeping OSS Flames single-tenant with no built-in user authentication. This is a deliberate boundary, not a gap we'll fill later in the same layer.

## The Approach

The OSS core will focus on VM orchestration, not identity. Operators will handle perimeter security — localhost-only, private network, reverse proxy, API gateway, VPN, or identity-aware proxy. We won't expose the OSS API directly to the public internet.

## What We're Leaving Out of OSS

No user accounts, no bearer-token auth, no RBAC, no organization ownership, no billing-aware permissions. All of that will go in `flames-platform`.

## What the Hosted Layer Will Add

User authentication, organizations, ownership checks, role-based access, billing integration — all in `flames-platform`, cleanly separated from the core.

## Audit Implications to Plan For

Without app-level auth, audit records won't always include a user actor. Operator-side gateways should inject request identity. Hosted deployments will enrich audit data through the platform layer.

## Why This Split

Keeps OSS simpler, easier to self-host, and focused on what it's good at. Identity is a complex domain — it deserves its own layer rather than being bolted onto the orchestrator.
