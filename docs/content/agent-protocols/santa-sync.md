---
sidebar_position: 3
title: Santa Sync
description: Configure Santa to sync with Woodstar.
---

# Santa Sync

Woodstar implements Santa's SyncV1 protocol over HTTPS with gzipped protobuf messages.

## Configure Santa

Create a Santa secret under **Enrollments > Santa**, then copy the generated configuration profile. The profile sets:

- `SyncBaseURL` to `<WOODSTAR_URL>/santa/sync/`
- Protobuf transfer and gzip encoding
- The Santa secret in the `Authorization` header

Leave `MachineID` unset so Santa uses the hardware UUID also reported by Orbit or osquery.

## Sync routes

| Method | Path                                    | Purpose                                 |
| ------ | --------------------------------------- | --------------------------------------- |
| `POST` | `/santa/sync/preflight/{machine_id}`    | Record client state and prepare rules   |
| `POST` | `/santa/sync/eventupload/{machine_id}`  | Upload execution and file-access events |
| `POST` | `/santa/sync/ruledownload/{machine_id}` | Download rules                          |
| `POST` | `/santa/sync/postflight/{machine_id}`   | Confirm the applied rule set            |

Santa must match an existing host. Preflight updates the Santa version, client mode, user details, SIP state, and rule-sync state. Rule downloads are paged in batches of 500.

Woodstar supports SyncV1. Santa SyncV2 and the associated push infrastructure are not implemented.

See [Santa](../admin/santa) for configurations, rules, and events.
