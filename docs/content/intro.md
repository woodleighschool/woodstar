---
sidebar_position: 1
title: Introduction
description: What Woodstar is, what this documentation covers, and what still needs owner knowledge.
---

# Woodstar

Woodstar is a self-hosted macOS observability and admin server. The code is built around Orbit/osquery enrollment first. Santa and Munki are native modules that add security policy, execution-event visibility, package metadata, and client repository responses when those parts are configured.

This documentation is a bootstrap pass. It is based on the repository state, not on a production rollout or hidden operating practice. Where the code is clear, the docs say so directly. Where a niche depends on real deployment choices, they stop short.

## Current Shape

- The backend is a Go server started by `cmd/woodstar`.
- Runtime configuration comes from `WOODSTAR_` environment variables and a small Cobra CLI.
- Postgres is the database. Migrations and sqlc live under `internal/database`.
- The admin UI is a React/Vite app under `web/`, served by the Go binary after build.
- The documented admin API is registered with Huma under `/api`.
- Agent-facing protocols are plain `chi` routes mounted outside the admin API contract.

## Capability Map

| Area | What It Owns |
| --- | --- |
| Hosts | Canonical Mac identity, enrollment metadata, host detail loading, user affinity, and cross-capability host enrichment. |
| Labels | Manual, dynamic osquery-backed, and derived directory labels used by checks, reports, Santa, and Munki scoping. |
| Agent secrets | Shared credentials accepted by agent-facing protocols. Orbit and osquery use them for enrollment. Santa and Munki use bearer-style protocol access. |
| Orbit | Orbit enrollment, node-key validation, config, device mapping, and Fleet-compatible placeholder endpoints that keep Orbit clients calm. |
| osquery | TLS-plugin enrollment, config, scheduled reports, checks, inventory projection, label evaluation, distributed writes, logs, and live queries. |
| Santa | Sync protocol, host state, configurations, rules, execution events, file-access events, and rule sync state. |
| Munki | Woodstar-managed software titles, pkginfo/package metadata, deployments, artifacts, manifest rendering, catalog rendering, and artifact redirects. |
| Directory | Optional Entra sync into users, groups, departments, and derived labels. |

## What Not To Read Into This

Woodstar is not documented here as a SaaS product, a polished public release, or a generic MDM. The current tree is a tool with sharp domain edges: osquery inventory, Santa policy sync, Munki repository behavior, and school-owned deployment assumptions that still need operator notes before a production runbook would be honest.

For development commands, start with [Local Development](./getting-started/local-development). For the server shape, read [Capability Boundaries](./concepts/capability-boundaries).
