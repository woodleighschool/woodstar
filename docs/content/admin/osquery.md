---
sidebar_position: 2
title: osquery
description: Scheduled reports, policies, remediation, and live queries.
---

# osquery

Woodstar uses osquery for inventory and SQL queries across enrolled Macs. Queries can run as reports, policies, dynamic labels, or one-off live queries.

## Reports

A report is a saved query with a schedule. Reports can run on all hosts or only hosts in selected labels. Woodstar stores the latest complete result snapshot for each targeted host, including empty observations, and shows hosts that have not reported yet as pending.

## Policies

A policy is a query that passes when at least one row is returned and fails when none are returned. A query error is recorded separately and does not change the host's last conclusive pass or fail state.

Use policies for conditions that should remain true, such as encryption being enabled or free disk space remaining above a threshold. Policies can include resolution guidance and an optional remediation script.

Automatic remediation runs once when an eligible host first fails or changes from passing to failing. Enabling automation, adding a script, or editing a script does not run it against hosts that are already failing. A successful script means only that the script exited with code 0; the policy remains failing until a later osquery evaluation passes.

Each host and policy keeps only its latest remediation run. Runs are not retried or redelivered automatically. **Run remediation again** explicitly queues the current script without changing the policy result. **Reset result** returns the result to pending; the next evaluation is treated as new and can trigger automatic remediation again.

Automatic and manual remediation require a host enrolled through Orbit with script execution enabled. See [Configure Orbit](../agent-protocols/orbit-and-osquery#configure-orbit).

## Live queries

A live query runs once against selected hosts or labels. Results appear as hosts respond and are not saved as a report definition.

The [Orbit and osquery](../agent-protocols/orbit-and-osquery) page covers enrollment and query transport.
