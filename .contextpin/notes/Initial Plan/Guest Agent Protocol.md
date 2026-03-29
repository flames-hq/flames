---
schema_version: 1
id: TGWUgQMJTg
created_at: 2026-03-29T15:59:14Z
updated_at: 2026-03-29T15:59:14Z
labels:
- DOCS
position: 2.0
---

# Guest Agent Plan

Every Flames VM will run a required guest agent — we're treating it as a core platform primitive from day one, not bolting it on later.

## What It Needs to Do

Read bootstrap metadata (vm_id, workload command, env, shutdown policy, bootstrap token) from MMDS or a config drive. Authenticate to the host-side agent over vsock. Start the configured workload. Report readiness, heartbeats, and exit status back to the controller.

## What It Won't Do

It won't be the source of truth for infrastructure state, won't schedule itself, and won't bypass host-side policy. The host orchestrator will own infra decisions; the agent owns in-guest behavior.

## Authentication Plan

We'll provision a per-VM vsock endpoint and a short-lived bootstrap token per VM. The guest agent will read the token from MMDS, connect over vsock, and authenticate. Tokens will be per-VM scoped and ideally single-use.

## Shutdown Flow

Host sends a shutdown request, agent stops the workload gracefully and acknowledges, host waits up to the configured grace period, then force kills if needed.

## First Version Scope

One primary workload per VM. We'll version the protocol from day one. Interactive exec and multi-process support will come later.
