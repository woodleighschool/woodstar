---
sidebar_position: 3
title: Santa Sync
description: Santa sync transport, endpoints, and current Woodstar behavior.
---

# Santa Sync

Santa sync uses protobuf payloads over HTTP. The current routes are mounted under `/santa/sync` and require an agent secret for `agent=santa`.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/santa/sync/preflight/{machine_id}` | Record Santa host observation, resolve configuration, prepare pending rule sync state. |
| `POST` | `/santa/sync/eventupload/{machine_id}` | Ingest execution and file-access events for an existing host. |
| `POST` | `/santa/sync/ruledownload/{machine_id}` | Return a page of pending Santa rules for the host. |
| `POST` | `/santa/sync/postflight/{machine_id}` | Promote pending sync state after the client reports received and processed rule counts. |

## Transport Requirements

The handler expects:

```http
Authorization: Bearer <santa-agent-secret>
Content-Type: application/x-protobuf
Content-Encoding: gzip
```

The request body limit is 16 MiB before decoding.

## Host Resolution

Santa resolves the host by `machine_id`. The service writes Santa host state only after it resolves an existing Woodstar host. It does not enroll a new host.

Preflight can update:

- Santa version
- reported client mode
- primary user and groups
- SIP status
- OS build
- model identifier
- rule sync counters

Event upload persists execution events and file-access events. Rule download returns up to 500 pending rules per page.

## Admin Resources

Santa admin routes expose configurations, rules, event lists, file-access event lists, and per-host rule state under `/api/santa` and `/api/hosts/{id}/santa/...`.
