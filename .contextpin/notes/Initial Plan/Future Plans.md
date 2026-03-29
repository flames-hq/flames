---
schema_version: 1
id: uIckPAXFHA
created_at: 2026-03-29T15:59:16Z
updated_at: 2026-03-29T15:59:16Z
labels:
- DOCS
position: 5.0
---

# Future Plans

Things we'll need eventually but are intentionally keeping out of the first implementation. These should shape today's architecture without being built yet.

## What's on the Horizon

**Policy model** — launch admission, runtime, ingress, egress, storage, and quota policies. We'll need these before opening up to external users.

**Abuse controls** — per-account quotas, max concurrent VMs, rate limits on creation, network abuse detection, rapid kill-switch isolation.

**Egress governance** — graduated controls from `deny` to `allow` to allowlisted destinations to shared NAT to dedicated public egress IP.

**Ingress governance** — no public ingress by default, authenticated ingress, per-endpoint policy, branded endpoints under `flames.sh`, custom domains.

**Files and backups** — VM file exports, user backup catalogs, snapshot-to-backup promotion, retention rules, content validation.

**Compliance and audit** — longer retention, exportable audit trails, actor attribution, policy-decision logging, incident response workflows.

## Why We're Waiting

These are platform-governance features, not minimum runtime-enablement. We need to nail correct isolation, reproducible VM lifecycle, clean storage abstractions, and a stable guest-agent contract first. But we're designing the hooks now so these can plug in later.
