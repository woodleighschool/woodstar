---
sidebar_position: 3
title: Agent Secrets
description: Shared secrets accepted by Woodstar's agent-facing protocols.
---

# Agent Secrets

Agent secrets are admin-managed shared credentials used by agent-facing protocols. They are stored through the `agentauth` package and managed through `/api/agent-secrets`.

The current agent values are:

| Agent | Used By |
| --- | --- |
| `orbit` | Orbit enrollment and osquery enrollment. |
| `santa` | Santa sync bearer authorization. |
| `munki` | Munki repository bearer authorization. |

The store requires secret values with at least 32 characters when creating or updating through the admin API.

## Enrollment Secret vs Node Key

Orbit and osquery use a shared agent secret at enrollment time. Successful enrollment issues a node key and stores that node key on the host. Follow-up Orbit/osquery requests use the node key, not the shared enrollment secret.

Re-enrolling the same host refreshes the node key. The old node key stops authenticating.

## Bearer Secrets

Santa and Munki protocol routes parse the `Authorization` header as a bearer token:

```http
Authorization: Bearer <secret>
```

The helper rejects empty tokens and tokens with whitespace. The token must verify against the matching agent value in the agent secret store.

## Admin Access

The agent secret admin resource is admin-only. Ordinary authenticated users can use the admin UI resources they are allowed to access, but creating, editing, or deleting shared agent protocol credentials is protected by admin middleware.
