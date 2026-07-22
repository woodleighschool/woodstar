---
sidebar_position: 2
title: osquery
description: Scheduled reports, checks, and live queries.
---

# osquery

Woodstar uses osquery for inventory and SQL queries across enrolled Macs. Queries can run as reports, checks, dynamic labels, or one-off live queries.

## Reports

A report is a saved query with a schedule. Reports can run on all hosts or only hosts in selected labels. Woodstar stores each result as a snapshot so changes can be reviewed over time.

## Checks

A check is a query that passes when at least one row is returned and fails when none are returned. Woodstar records the latest result for each host and shows the passing and failing counts.

Use checks for conditions that should remain true, such as encryption being enabled or a required process running.

## Live queries

A live query runs once against selected hosts or labels. Results appear as hosts respond and are not saved as a report definition.

The [Orbit and osquery](../agent-protocols/orbit-and-osquery) page covers enrollment and query transport.
