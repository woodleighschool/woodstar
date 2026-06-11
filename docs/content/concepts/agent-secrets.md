---
sidebar_position: 3
title: Agent Secrets
description: The shared credentials the agent protocols accept, and how they differ from node keys.
---

# Agent Secrets

Agent secrets are the shared credentials the Mac clients use to talk to Woodstar. You manage them in the admin app under Enrollments, and they're stored separately from user passwords and API keys.

There are three:

| Secret | Used by |
| --- | --- |
| `orbit` | Orbit and osquery enrollment. |
| `santa` | Santa sync. |
| `munki` | Munki repository access. |

Each value has to be at least 32 characters.

## Enrollment secrets and node keys

Orbit and osquery use the shared secret only at enrollment. Enrolling successfully mints a *node key*, which Woodstar stores on the host, and every request after that carries the node key instead of the shared secret. Re-enrolling the same Mac issues a fresh node key and retires the old one.

So the shared secret is the thing you hand out when setting up a machine. The node key is the per-host credential it earns once it's in.

## Bearer secrets

Santa and Munki are simpler. Their requests carry the shared secret straight through as a bearer token:

```http
Authorization: Bearer <secret>
```

Empty tokens and tokens with whitespace are rejected, and the token has to match the secret for that agent.

## Who can manage them

Creating, editing, and deleting agent secrets is admin-only. A regular Woodstar user can work in the parts of the app they're allowed into, but the shared protocol credentials sit behind admin access.
