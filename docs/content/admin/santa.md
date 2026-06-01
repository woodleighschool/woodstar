---
sidebar_position: 3
title: Santa
description: Santa configurations, rules, events, and host rule state.
---

# Santa

Santa adds execution policy and event visibility for hosts that already exist in Woodstar. It is optional: the core app still works without Santa sync traffic.

## Configurations

Configurations are ordered. They can target labels and carry Santa client policy settings such as client mode, bundle support, transitive rules, event upload, sync interval, batch size, path regexes, removable-media policy, and event detail links.

| Route | Purpose |
| --- | --- |
| `GET /api/santa/configurations` | List configurations. |
| `POST /api/santa/configurations` | Create a configuration. |
| `GET /api/santa/configurations/{id}` | Load one configuration. |
| `PATCH /api/santa/configurations/{id}` | Update one configuration. |
| `DELETE /api/santa/configurations/{id}` | Delete one configuration. |
| `POST /api/santa/configurations/bulk-delete` | Delete multiple configurations. |
| `PUT /api/santa/configurations/order` | Reorder configurations. |

Label conflicts return `409` when a label is already assigned to another configuration.

## Rules

Rules model Santa targets and policies. The current rule types are `binary`, `certificate`, `teamid`, `signingid`, `cdhash`, and `bundle`. Policies include allowlist, compiler allowlist, blocklist, silent blocklist, and CEL.

| Route | Purpose |
| --- | --- |
| `GET /api/santa/rules` | List rules. |
| `GET /api/santa/rule-targets` | Search target candidates from observed execution data. |
| `POST /api/santa/rules` | Create a rule. |
| `GET /api/santa/rules/{id}` | Load one rule. |
| `PATCH /api/santa/rules/{id}` | Update one rule. |
| `DELETE /api/santa/rules/{id}` | Delete one rule. |
| `POST /api/santa/rules/bulk-delete` | Delete multiple rules. |
| `PUT /api/santa/rules/{id}/includes/order` | Reorder rule includes. |
| `GET /api/hosts/{id}/santa/rules` | List effective rule state for one host. |

Rule includes attach a policy to a label. Exclude label IDs remove hosts from a rule even when an include matches.

## Events

Santa event routes are read-only in the admin API:

- `GET /api/santa/events`
- `GET /api/santa/events/{id}`
- `GET /api/santa/file-access-events`
- `GET /api/santa/file-access-events/{id}`

Execution events include host summary, executable metadata, user/session fields, decision, and timestamps. File-access events use Santa's file-access decision values.

Retention cleanup is controlled by `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` and `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL`.
