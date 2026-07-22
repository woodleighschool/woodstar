---
sidebar_position: 3
title: Santa
description: Santa configurations, rules, and events.
---

# Santa

Woodstar provides the sync server for [Santa](https://github.com/northpolesec/santa). Macs download their configuration and rules, then upload execution and file-access events.

## Configurations

A configuration contains Santa client settings and targets one or more labels. Configurations are ordered, and a label can belong to only one configuration.

Settings include client mode, sync timing, event upload, bundle handling, path regular expressions, and removable-media policy.

## Rules

Rules allow or block binaries, certificates, team IDs, signing IDs, code-directory hashes, or bundles. Each rule includes one label and can exclude other labels.

The rule form can search observed execution events for known targets. A host page shows the rules that apply to that Mac.

## Events

Woodstar stores execution events and file-access events reported by Santa. The event pages are read-only.

`WOODSTAR_SANTA_EVENT_RETENTION_DAYS` controls retention. See [Environment](../configuration/environment#santa-event-retention).

See [Santa Sync](../agent-protocols/santa-sync) for client configuration.
