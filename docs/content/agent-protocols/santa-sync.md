---
sidebar_position: 3
title: Santa Sync
description: The Santa sync transport, its four stages, and how rules page out to clients.
---

# Santa Sync

Santa syncs over HTTPS using protobuf payloads. The routes live under `/santa/sync` and authenticate with the `santa` agent secret. Each request is for one machine, named by its `machine_id` in the path.

## The four stages

Santa walks through the same four-stage handshake every sync.

| Method | Path                                    | Purpose                                                                                                    |
| ------ | --------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `POST` | `/santa/sync/preflight/{machine_id}`    | Record the host's Santa state, resolve which configuration applies, and stage the rules waiting to go out. |
| `POST` | `/santa/sync/eventupload/{machine_id}`  | Take execution, file-access, and standalone rule-creation events for the host.                             |
| `POST` | `/santa/sync/ruledownload/{machine_id}` | Hand back a page of pending rules.                                                                         |
| `POST` | `/santa/sync/postflight/{machine_id}`   | Promote the staged rules once the client reports how many it received and applied.                         |

## Transport

The handler expects:

```http
Authorization: Bearer <santa-agent-secret>
Content-Type: application/x-protobuf
Content-Encoding: gzip
```

The body limit is 16 MiB before decoding.

Configure Santa with a device configuration profile whose payload type is `com.northpolesec.santa`. The relevant settings are:

- `SyncBaseURL`: `https://woodstar.example/santa/sync/`
- `SyncEnableProtoTransfer`: `true`
- `SyncClientContentEncoding`: `gzip`
- `SyncExtraHeaders`: a dictionary containing `Authorization: Bearer <santa-agent-secret>`

Santa requires HTTPS except for a loopback sync server. A private development CA must also be trusted by macOS; `ServerAuthRootsFile` pins the expected root but does not satisfy App Transport Security by itself.

## Host resolution

Santa finds the host by `machine_id`, and it only writes Santa state once it has matched an existing Woodstar host. It never enrolls a new one. If the machine is unknown, there's nowhere to put the state.

Santa's default `MachineID` is the hardware UUID reported to Woodstar by Orbit or osquery, so Woodstar uses that shared identifier for Santa host resolution.

Preflight is the catch-up moment. It can update the host's Santa version, reported client mode, primary user and groups, SIP status, and rule sync counters. Orbit and osquery remain the owners of host hardware and operating-system inventory.

Event upload persists execution and file-access events, including execution `static_rule`, plus syncv1 standalone rule-creation audit events. Unknown audit-event variants are rejected rather than acknowledged and discarded.

Rule download pages out up to 500 pending rules at a time. A clean sync sends the full desired set. A normal sync sends only new or changed rules and removals. Postflight must report the exact sync type and both received and processed counts must match the pending payload before Woodstar marks the desired snapshot applied.

Santa's rule hash is client-owned and opaque. Woodstar records the preflight and confirmed postflight hashes, uses a later mismatch to request a clean sync, and requires the hash to remain unchanged when an incremental sync has no rule payload.

Woodstar implements the current syncv1 protobuf flow. It does not issue the trusted push-token chain or run the NATS infrastructure required by SyncV2. Santa settings that Woodstar does not expose, including `disable_unknown_event_upload` and `override_file_access_action`, keep their Santa defaults.

The admin side of Santa (configurations, rules, events, and per-host rule state) is in [Santa](../admin/santa).
