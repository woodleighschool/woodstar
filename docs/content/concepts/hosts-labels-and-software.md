---
sidebar_position: 2
title: Hosts, Labels, and Software
description: The records used throughout Woodstar.
---

# Hosts, Labels, and Software

Hosts and labels connect inventory with the settings sent to each Mac. Software appears in two places because observed and managed software answer different questions.

## Hosts

A host is an enrolled Mac. Woodstar records its identity, hardware, operating system, network details, storage, agent versions, users, and last-seen time.

The host page also shows related data such as labels, certificates, batteries, software, query results, Santa state, and Munki state when available.

## Labels

Labels group hosts for targeting.

| Type      | Membership                                 |
| --------- | ------------------------------------------ |
| `manual`  | Hosts selected by hand                     |
| `dynamic` | Hosts that match an osquery query          |
| `derived` | Hosts that match directory users or groups |

Woodstar also creates the built-in **All Hosts** label.

## Observed and managed software

The **Software** inventory is populated by osquery and shows what is installed, including versions, paths, bundle identifiers, browser extensions, and signing data.

The **Munki** software library describes what should be installed or offered. A title can have several package versions and label-based assignments.

## User affinity

User affinity links a host to a directory user. Woodstar can infer the person from Orbit and Santa data, while the host page allows manual changes. Directory sync adds names, departments, and group membership to the user record.

User affinity does not grant access to Woodstar.
