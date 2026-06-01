---
sidebar_position: 1
title: Capability Boundaries
description: How Woodstar's backend packages are split and wired.
---

# Capability Boundaries

Woodstar's backend is organized by product capability, with `cmd/woodstar/main.go` acting as the wiring point. That file loads config, opens the database, builds stores and services, mounts routes, starts background loops, and starts the HTTP server.

The admin HTTP surface is deliberately separate from the agent protocol surface.

## Admin API

Admin endpoints are Huma routes registered under `/api`. They live in `internal/api/handlers`, even when a domain package owns the resource model and store.

Examples:

- `/api/hosts`
- `/api/labels`
- `/api/osquery/reports`
- `/api/osquery/checks`
- `/api/santa/rules`
- `/api/munki/software-titles`

This keeps the SPA contract in one place. Domain packages do not host admin handlers.

## Agent Protocols

Agent-facing routes speak the protocol shape expected by the client. They are mounted with chi, not Huma:

| Capability | Package | Route Family |
| --- | --- | --- |
| Orbit | `internal/orbit/protocol` | `/api/fleet/orbit/...` |
| osquery | `internal/osquery/protocol` | `/api/osquery/...` and `/api/v1/osquery/...` |
| Santa | `internal/santa/protocol` | `/santa/sync/...` |
| Munki | `internal/munki/protocol` | `/munki/...` |

These routes are not public admin API resources. They validate node keys, shared agent secrets, headers, and wire formats according to their own client contract.

## Runtime Ownership

The `api.Dependencies` struct groups runtime services by capability:

- `Runtime`: config, database, version, logger, sessions, and the embedded web handler.
- `Auth`: local accounts, sessions, OIDC, and account API key operations.
- `Inventory`: hosts, user affinity, software inventory, and labels.
- `AgentAuth`: shared agent secret store.
- `Orbit`, `Osquery`, `Santa`, `Munki`: protocol and admin resources for each capability.
- `Directory`: optional Entra-backed users, groups, departments, and derived label data.

The main function builds those dependencies explicitly. There is no hidden application container.

## Cross-Capability Data

Hosts are the anchor. Orbit and osquery can create or refresh hosts during enrollment. Santa and Munki resolve existing hosts and enrich them; they do not create host rows from scratch.

Labels are the shared targeting primitive. osquery reports, osquery checks, Santa configurations, Santa rules, and Munki deployments all rely on label-based scoping in some form. The labels package stays independent of those capabilities.
