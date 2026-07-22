---
sidebar_position: 1
title: Introduction
description: What Woodstar does and how its parts fit together.
---

# Woodstar

Woodstar is self-hosted macOS management for the gaps Intune leaves. Woodstar provides:

- **Managed software** through Munki.
- **Execution rules and events** through Santa.
- **Inventory, reports, checks, and live queries** through osquery.
- **Host enrollment** through Orbit or osquery.

## Clients

| Client  | What the client uses Woodstar for                            |
| ------- | ------------------------------------------------------------ |
| Orbit   | Enrollment and basic host details                            |
| osquery | Inventory, reports, checks, labels, and live queries         |
| Santa   | Client configuration, rules, and execution events            |
| Munki   | Software assignments, repository data, and package downloads |

Orbit and osquery can enroll a Mac. Santa and Munki connect to an existing host using its hardware identity.

Labels group hosts. The same labels can scope osquery work, Santa policy, and Munki software.

Woodstar implements the Fleet-compatible endpoints used by Orbit and osquery. Santa and Munki use their own native protocols.

## Documentation

- [Getting Started](./getting-started/docker-compose) runs Woodstar with Docker Compose.
- [Concepts](./concepts/capability-boundaries) explains hosts, labels, accounts, and agent credentials.
- [Using Woodstar](./admin/hosts-and-inventory) covers the main areas of the web app.
- [Agent Protocols](./agent-protocols/overview) documents client configuration and routes.
- [AutoPkg](./autopkg/overview) covers package imports.
- [Configuration](./configuration/environment) lists server settings.
- [Development](./development/setup) covers a source checkout and repository commands.
- [API Reference](./api/overview) documents the JSON API used by the web app and scripts.
