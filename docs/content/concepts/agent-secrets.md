---
sidebar_position: 3
title: Agent Secrets
description: Credentials used by Orbit, osquery, Santa, and Munki.
---

# Agent Secrets

Agent secrets authenticate Mac clients. Create the shared Orbit/osquery secret from **Hosts > Enroll Hosts** or the **osquery** overview. Santa and Munki secrets live on their integration overviews.

| Secret  | Used by                      |
| ------- | ---------------------------- |
| `orbit` | Orbit and osquery enrollment |
| `santa` | Santa sync                   |
| `munki` | Munki repository requests    |

Each secret must be at least 32 characters.

## Orbit and osquery

Orbit and osquery send the shared enrollment secret once. The server returns a node key for later requests. The other protocol's first enrollment attaches to the same host. Re-enrollment retains the host identity and dependent state while replacing that protocol's node key.

## Santa and Munki

Santa and Munki send their shared secret as a bearer token:

```http
Authorization: Bearer <secret>
```

Agent secrets are separate from account passwords and API keys.
