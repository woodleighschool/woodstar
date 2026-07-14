---
sidebar_position: 2
title: Orbit and osquery
description: Enrollment, config, distributed queries, and the Fleet-compatible endpoints.
---

# Orbit and osquery

Orbit and osquery are two separate enrolling clients. Both authenticate with a shared enrollment secret the first time, then switch to their own per-host node key for everything after. Orbit speaks the Fleet Orbit protocol, so an off-the-shelf Orbit client works against Woodstar.

Both clients connect to the HTTPS `WOODSTAR_URL`. Development clients must trust the local mkcert CA; Woodstar does not generate insecure Orbit or osquery settings.

## Orbit endpoints

| Method | Path                              | Purpose                                                                                 |
| ------ | --------------------------------- | --------------------------------------------------------------------------------------- |
| `POST` | `/api/fleet/orbit/enroll`         | Check the enroll secret, upsert the host by hardware UUID, and issue an Orbit node key. |
| `POST` | `/api/fleet/orbit/config`         | Validate the node key and return the host's Orbit config.                               |
| `PUT`  | `/api/fleet/orbit/device_mapping` | Validate the node key and record an email from the device profile as user affinity.     |
| `HEAD` | `/api/fleet/orbit/ping`           | Return `200 OK`.                                                                        |

Orbit clients expect a few more endpoints to exist even when the feature behind them isn't something Woodstar does, so Woodstar answers the Fleet-compatible paths (scripts, software install, setup experience, device token, disk encryption, LUKS) with empty responses. They keep the client happy; they aren't features.

Orbit responses advertise their capabilities:

```http
X-Fleet-Capabilities: orbit_endpoints,end_user_email
```

## osquery endpoints

Each osquery endpoint is mounted under both `/api/osquery` and `/api/v1/osquery`.

| Method | Path                 | Purpose                                                                                          |
| ------ | -------------------- | ------------------------------------------------------------------------------------------------ |
| `POST` | `/enroll`            | Check the enroll secret, parse the host details, upsert the host, and issue an osquery node key. |
| `POST` | `/config`            | Return the scheduled query config and osquery options.                                           |
| `POST` | `/distributed/read`  | Hand the host its queued detail queries, label queries, checks, and live queries.                |
| `POST` | `/distributed/write` | Take the results back.                                                                           |
| `POST` | `/log`               | Accept osquery logs and scheduled report results.                                                |
| `POST` | `/carve/begin`       | Return an empty object.                                                                          |
| `POST` | `/carve/block`       | Return an empty object.                                                                          |

A bad node key doesn't break the client. The osquery service returns `node_invalid: true` where the TLS plugin expects it, and the client re-enrolls.

## What distributed writes do

The write endpoint is where most of the inventory work happens. A single batch of results can:

- update the host's detail and inventory
- project observed software
- evaluate dynamic labels
- record check pass/fail membership
- save report result snapshots
- return live-query results to the in-memory live-query manager

If a host is missing the Orbit extension tables some queries want, that's just query result state, not a separate mode the server runs in.
