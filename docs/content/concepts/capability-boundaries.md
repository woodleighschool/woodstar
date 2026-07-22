---
sidebar_position: 1
title: How Woodstar Fits Together
description: The web app, agent protocols, hosts, and labels.
---

# How Woodstar Fits Together

Woodstar serves the web app and the Mac agent protocols from the same process. They share host and policy data but use different routes and credentials.

## Web app and API

The React app uses the JSON API under `/api`. A signed-in account uses a session cookie; scripts can use an account API key. The [API reference](../api/overview) covers this surface.

## Agent protocols

Orbit, osquery, Santa, and Munki connect to separate route families that match each client protocol. They use agent secrets or per-host node keys instead of browser sessions. See [Agent Protocols](../agent-protocols/overview).

## Hosts

A host is an enrolled Mac. Its hardware, operating system, users, software, query results, Santa state, and Munki state all attach to the same record.

Orbit and osquery can create or refresh a host during enrollment. Santa and Munki require that host to exist first.

## Labels

Labels group hosts for targeting.

| Type    | Membership                                               |
| ------- | -------------------------------------------------------- |
| Manual  | Hosts selected in Woodstar                               |
| Dynamic | Hosts that match an osquery query                        |
| Derived | Hosts that match directory data, such as a group or user |

Labels can scope osquery reports and checks, Santa configurations and rules, and Munki software.
