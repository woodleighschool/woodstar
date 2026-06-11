---
sidebar_position: 1
title: Overview
description: The agent-facing route families and how they differ from the admin API.
---

# Agent Protocols

The Macs don't use the admin API. They talk to a separate set of endpoints, each shaped to match the client that's calling: Orbit, osquery, Santa, and Munki. These routes sit beside the admin API but they're a different surface, with different authentication and different wire formats.

| Client | Route family | Auth |
| --- | --- | --- |
| Orbit | `/api/fleet/orbit/...` | Enroll secret, then an Orbit node key. |
| osquery | `/api/osquery/...`, `/api/v1/osquery/...` | Enroll secret, then an osquery node key. |
| Santa | `/santa/sync/...` | Bearer token for the `santa` secret. |
| Munki | `/munki/...` | Bearer token for the `munki` secret, plus a `Serial` header. |

The split between "creates a host" and "attaches to a host" matters here. Orbit and osquery can create or refresh a host record while enrolling. Santa and Munki resolve a host that already exists, and if it doesn't, they return the protocol's own not-found shape rather than enrolling anything. See [Agent Secrets](../concepts/agent-secrets) for how node keys differ from the shared enrollment secrets.

The pages in this section describe each protocol's endpoints and the behaviour that isn't obvious from the route alone. The admin-facing endpoints (the ones the SPA uses) are in the [API reference](../api/overview) instead.
