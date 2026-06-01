---
sidebar_position: 2
title: osquery
description: Reports, checks, live queries, and their admin API routes.
---

# osquery

Woodstar uses osquery for scheduled report snapshots, pass/fail checks, live queries, dynamic labels, and inventory projection.

## Reports

Reports are saved osquery SQL definitions with a schedule interval and optional label scope.

| Route | Purpose |
| --- | --- |
| `GET /api/osquery/reports` | List reports. |
| `POST /api/osquery/reports` | Create a report. |
| `GET /api/osquery/reports/{id}` | Load one report. |
| `PUT /api/osquery/reports/{id}` | Replace report settings. |
| `DELETE /api/osquery/reports/{id}` | Delete one report. |
| `POST /api/osquery/reports/bulk-delete` | Delete multiple reports. |
| `GET /api/osquery/reports/{id}/results` | List saved result snapshots. |
| `GET /api/hosts/{id}/osquery/reports` | List reports for one host. |
| `GET /api/hosts/{id}/osquery/reports/{report_id}` | Load one host report. |

`schedule_interval` accepts non-negative values. The code does not document a scheduler guarantee beyond the intervals it sends through osquery config.

## Checks

Checks are query-backed pass/fail rules. A check has a SQL query, optional label scope, and aggregate pass/fail host counts.

| Route | Purpose |
| --- | --- |
| `GET /api/osquery/checks` | List checks. |
| `POST /api/osquery/checks` | Create a check. |
| `GET /api/osquery/checks/{id}` | Load one check. |
| `PUT /api/osquery/checks/{id}` | Replace check settings. |
| `DELETE /api/osquery/checks/{id}` | Delete one check. |
| `POST /api/osquery/checks/bulk-delete` | Delete multiple checks. |
| `GET /api/osquery/checks/{id}/hosts` | List host pass/fail state for one check. |
| `GET /api/hosts/{id}/osquery/checks` | List checks for one host. |

The check query must return enough signal for Woodstar to interpret a pass/fail state. The exact authoring convention should stay close to the frontend editor and backend check tests as that area evolves.

## Live Queries

Live queries are command-shaped. They do not behave like persisted report definitions.

| Route | Purpose |
| --- | --- |
| `POST /api/live-queries` | Start a live query. |
| `POST /api/live-queries/targets/count` | Count matching target hosts. |
| `POST /api/live-queries/{id}/stop` | Stop a live query. |
| `GET /api/live-queries/{id}/stream` | Stream live query events. |

Results come back through osquery distributed writes and are tracked by the in-memory live-query manager.
