---
sidebar_position: 3
title: Santa Sync
description: The Santa sync transport, its four stages, and how rules page out to clients.
---

# Santa Sync

Santa syncs over HTTP using protobuf payloads. The routes live under `/santa/sync` and authenticate with the `santa` agent secret. Each request is for one machine, named by its `machine_id` in the path.

## The four stages

Santa walks through the same four-stage handshake every sync.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/santa/sync/preflight/{machine_id}` | Record the host's Santa state, resolve which configuration applies, and stage the rules waiting to go out. |
| `POST` | `/santa/sync/eventupload/{machine_id}` | Take execution and file-access events for the host. |
| `POST` | `/santa/sync/ruledownload/{machine_id}` | Hand back a page of pending rules. |
| `POST` | `/santa/sync/postflight/{machine_id}` | Promote the staged rules once the client reports how many it received and applied. |

## Transport

The handler expects:

```http
Authorization: Bearer <santa-agent-secret>
Content-Type: application/x-protobuf
Content-Encoding: gzip
```

The body limit is 16 MiB before decoding.

## Host resolution

Santa finds the host by `machine_id`, and it only writes Santa state once it has matched an existing Woodstar host. It never enrolls a new one. If the machine is unknown, there's nowhere to put the state.

Preflight is the catch-up moment. It can update the host's Santa version, reported client mode, primary user and groups, SIP status, OS build, model identifier, and the rule sync counters.

Event upload persists both execution and file-access events. Rule download pages out up to 500 pending rules at a time, which is why postflight exists: the client confirms what it got, and only then does Woodstar mark those rules delivered.

The admin side of Santa (configurations, rules, events, and per-host rule state) is in [Santa](../admin/santa).
