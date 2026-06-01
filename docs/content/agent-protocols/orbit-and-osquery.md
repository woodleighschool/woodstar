---
sidebar_position: 2
title: Orbit And osquery
description: Enrollment, config, distributed query, and compatibility endpoints for Orbit and osquery clients.
---

# Orbit And osquery

Orbit and osquery are separate enrolling clients. Both use shared agent secrets during enrollment, then use their own host node keys afterward.

## Orbit Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/fleet/orbit/enroll` | Validate the Orbit enroll secret, upsert the host by hardware UUID, and issue an Orbit node key. |
| `POST` | `/api/fleet/orbit/config` | Validate the Orbit node key and return the current Orbit config. |
| `PUT` | `/api/fleet/orbit/device_mapping` | Validate the Orbit node key and record a profile-provided email as user affinity. |
| `HEAD` | `/api/fleet/orbit/ping` | Return `200 OK`. |

Woodstar also returns Fleet-compatible empty responses for several Orbit paths such as scripts, software install, setup experience, device token, disk encryption, and LUKS data. These are compatibility endpoints, not Woodstar feature promises.

Orbit responses set:

```http
X-Fleet-Capabilities: orbit_endpoints,end_user_email
```

## osquery Endpoints

Each osquery endpoint is mounted under both `/api/osquery` and `/api/v1/osquery`.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/enroll` | Validate the enroll secret, parse host details, upsert the host, and issue an osquery node key. |
| `POST` | `/config` | Return scheduled query config and osquery options. |
| `POST` | `/distributed/read` | Queue detail queries, label queries, checks, and live queries for the host. |
| `POST` | `/distributed/write` | Ingest distributed query results. |
| `POST` | `/log` | Accept osquery logs and scheduled report results. |
| `POST` | `/carve/begin` | Return an empty JSON object. |
| `POST` | `/carve/block` | Return an empty JSON object. |

Invalid node keys do not crash the client path. The osquery service returns `node_invalid: true` where the TLS plugin expects it.

## Inventory Work

osquery distributed writes do several jobs:

- update host detail inventory
- project observed software
- evaluate dynamic labels
- record check pass/fail membership
- save report result snapshots
- return live-query results to the in-memory live-query manager

Missing Orbit extension data should be treated as query result state. It is not a separate server mode.
