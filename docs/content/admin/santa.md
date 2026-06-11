---
sidebar_position: 3
title: Santa
description: "Execution policy and event visibility: configurations, rules, and events."
---

# Santa

Santa decides which binaries are allowed to run on a Mac, and reports back what it sees. Woodstar is the sync server: you write the policy here, the Macs pull it down, and the execution events come back for you to look at.

Santa is optional. A host works fine without it; the Santa sections just stay empty until that host starts syncing.

## Configurations

A configuration is the client policy Santa runs under: client mode, bundle support, transitive rules, event upload, sync interval, batch size, path regexes, removable-media policy, and the link Santa uses for event detail. Configurations are ordered and target labels.

A label can only belong to one configuration. Assign a label that's already taken and Woodstar returns a conflict rather than letting two configurations fight over the same hosts.

## Rules

Rules are the allow and block decisions. Each rule has a target type and a policy.

Target types: `binary`, `certificate`, `teamid`, `signingid`, `cdhash`, and `bundle`. Policies: allowlist, compiler allowlist, blocklist, silent blocklist, and CEL.

A rule attaches its policy to a label through an include, and you can layer excludes to carve hosts back out even when an include would otherwise match. When you're hunting for something to write a rule about, the rule-target search pulls candidates straight from observed execution data, so you can build a rule from a binary you've actually seen run.

You can also look at the effective rule state for a single host, which is the set of rules that machine should be enforcing.

## Events

Events are read-only here. Santa reports two kinds:

- **Execution events**: what tried to run, with the executable's metadata, the user and session, the decision, and timestamps.
- **File-access events**: access against the paths Santa is watching, with Santa's file-access decision values.

Events accumulate, so they're swept on a schedule. Retention is controlled by `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` and the sweep cadence by `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL` (see [Environment](../configuration/environment)).

The sync protocol itself, including how rules are paged out to clients, is in [Santa Sync](../agent-protocols/santa-sync). Endpoints are in the [API reference](../api/overview).
