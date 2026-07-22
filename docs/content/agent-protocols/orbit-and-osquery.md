---
sidebar_position: 2
title: Orbit and osquery
description: Enroll Macs and run osquery through Fleet-compatible endpoints.
---

# Orbit and osquery

Orbit and osquery enroll with the Orbit agent secret, then use a per-host node key. Woodstar supports the Fleet route and response shapes used by both clients.

## Configure Orbit

Under **Enrollments > Orbit**, create an agent secret and copy the generated package command and configuration profile. The profile sets the Woodstar URL, enrollment secret, and optional MDM user-email value.

## Orbit routes

| Method | Path                                    | Purpose                        |
| ------ | --------------------------------------- | ------------------------------ |
| `POST` | `/api/fleet/orbit/enroll`               | Enroll and return a node key   |
| `POST` | `/api/fleet/orbit/config`               | Return the Orbit configuration |
| `PUT`  | `/api/fleet/orbit/device_mapping`       | Record the assigned user email |
| `POST` | `/api/fleet/orbit/device_token`         | Rotate the device token        |
| `HEAD` | `/api/fleet/orbit/ping`                 | Check the server               |
| `HEAD` | `/api/latest/fleet/device/{token}/ping` | Validate a device token        |

## osquery routes

| Method | Path                                | Purpose                            |
| ------ | ----------------------------------- | ---------------------------------- |
| `POST` | `/api/v1/osquery/enroll`            | Enroll and return a node key       |
| `POST` | `/api/v1/osquery/config`            | Return schedule and client options |
| `POST` | `/api/v1/osquery/distributed/read`  | Return queued queries              |
| `POST` | `/api/v1/osquery/distributed/write` | Receive distributed-query results  |
| `POST` | `/api/v1/osquery/log`               | Receive scheduled report results   |

An invalid node key tells osquery to enroll again. Woodstar does not enable file carving or status-log forwarding.

Distributed results update host details, software inventory, dynamic labels, checks, reports, and active live queries.
