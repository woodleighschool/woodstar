---
sidebar_position: 1
title: Overview
description: Agent-facing route families and how they differ from the admin API.
---

# Agent Protocols

Woodstar mounts agent protocols beside the admin API, but they are not the same surface.

Admin routes are session-authenticated JSON resources under `/api`. Agent routes are shaped around existing clients: Orbit, osquery, Santa, and Munki.

| Client | Route Family | Auth Shape |
| --- | --- | --- |
| Orbit | `/api/fleet/orbit/...` | enrollment secret, then Orbit node key |
| osquery | `/api/osquery/...`, `/api/v1/osquery/...` | enrollment secret, then osquery node key |
| Santa | `/santa/sync/...` | bearer token for `agent=santa` |
| Munki | `/munki/...` | bearer token for `agent=munki` plus `Serial` host header |

Orbit and osquery can create or refresh host rows during enrollment. Santa and Munki resolve existing hosts. If the host is unknown, those protocols return the protocol-level failure for missing state instead of creating a host.

## Source Files

| Protocol | Route Registration |
| --- | --- |
| Orbit | `internal/orbit/protocol/orbit.go` |
| osquery | `internal/osquery/protocol/osquery.go` |
| Santa | `internal/santa/protocol/santa.go` |
| Munki | `internal/munki/protocol/munki.go` |

The combined mount point is `internal/api/protocols.go`.
