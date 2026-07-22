---
sidebar_position: 1
title: Overview
description: Client routes and credentials for Orbit, osquery, Santa, and Munki.
---

# Agent Protocols

Woodstar exposes the native protocol expected by each Mac client. These routes are separate from the JSON API used by the web app.

| Client                   | Route family                  | Authentication                           |
| ------------------------ | ----------------------------- | ---------------------------------------- |
| Orbit                    | `/api/fleet/orbit/...`        | Enrollment secret, then Orbit node key   |
| osquery                  | `/api/v1/osquery/...`         | Enrollment secret, then osquery node key |
| Santa                    | `/santa/sync/...`             | Santa bearer secret                      |
| Munki                    | `/munki/...`                  | Munki bearer secret                      |
| Munki distribution point | `/api/munki/distribution/...` | Per-point key                            |

Create the Orbit, Santa, and Munki secrets under **Enrollments**. The same pages provide configuration-profile templates for each client.

Orbit and osquery can enroll a new host. Santa and Munki require a host that has already enrolled with the same hardware UUID or serial number.

See [Agent Secrets](../concepts/agent-secrets) for the credential model.
