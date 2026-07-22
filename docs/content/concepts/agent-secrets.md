---
sidebar_position: 3
title: Agent Secrets
description: Credentials used by Orbit, osquery, Santa, and Munki.
---

# Agent Secrets

Agent secrets authenticate Mac clients. Create them under **Enrollments** in Woodstar.

| Secret  | Used by                      |
| ------- | ---------------------------- |
| `orbit` | Orbit and osquery enrollment |
| `santa` | Santa sync                   |
| `munki` | Munki repository requests    |

Each secret must be at least 32 characters.

## Orbit and osquery

Orbit and osquery send the shared enrollment secret once. Woodstar returns a node key for later requests. Re-enrolling a host replaces its previous node key.

## Santa and Munki

Santa and Munki send their shared secret as a bearer token:

```http
Authorization: Bearer <secret>
```

Agent secrets are separate from account passwords and API keys.
