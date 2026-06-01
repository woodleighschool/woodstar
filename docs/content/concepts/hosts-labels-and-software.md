---
sidebar_position: 2
title: Hosts, Labels, And Software
description: The inventory objects most admin views build on.
---

# Hosts, Labels, And Software

Woodstar's admin UI mostly starts from three observed objects: hosts, labels, and software titles. Santa and Munki add their own views, but they still resolve back to hosts and labels.

## Hosts

A host is an enrolled Mac. The host row carries identity, hardware, OS, network, storage, agent versions, enrollment metadata, user affinity, and timestamps.

Host detail adds children:

- labels
- local users
- batteries
- certificates
- software inventory
- osquery report and check state
- Santa state when present
- Munki state when present

The current model exposes `display_name` as the host label used by the admin UI. Raw hostnames, computer names, serials, and UUIDs still exist as data, but they are not a cascade of alternate display names.

## Labels

Labels group hosts. The current label membership types are:

| Type | Meaning |
| --- | --- |
| `manual` | Admin-selected host IDs. |
| `dynamic` | osquery-backed membership using a label query. |
| `derived` | Directory-derived membership from attributes such as department, group, or user. |

Labels can be regular or builtin. Dynamic labels depend on osquery distributed query results. Derived labels depend on directory sync data.

## Software

`internal/software` is observed inventory. It is populated from osquery and stores titles, versions, install paths, browser extensions, bundle identifiers, signing data, and host counts.

Munki desired state is separate. Munki software titles and deployments live under `internal/munki`; osquery seeing installed packages is still observation and remains software inventory.

## User Affinity

User affinity combines host-to-user observations from several sources. Orbit can write an email from device mapping, Santa can write the primary user seen during preflight, and directory sync can enrich the resolved user with name, department, and groups.

The user affinity model is useful for filters and host detail, but it is not an authentication source.
