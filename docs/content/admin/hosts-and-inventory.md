---
sidebar_position: 1
title: Hosts and Inventory
description: Hosts, labels, software inventory, and directory data.
---

# Hosts and Inventory

The host list shows every enrolled Mac. Search and filters can narrow the list by status, label, software, or osquery policy result.

## Host details

A host page combines its hardware and operating-system details with users, batteries, certificates, software, labels, and osquery results. Agent-specific tabs appear after the corresponding agent contacts the server. Empty tables retain their columns and controls. The host header shows when inventory was fetched.

Use **Assigned User** to correct or clear the person associated with a Mac.

## Labels

Labels group hosts for reports, policies, Santa policy, and Munki software.

- **Manual** labels contain selected hosts.
- **Dynamic** labels use an osquery query.
- **Derived** labels use directory users, groups, or departments.

See [Hosts, Labels, and Software](../concepts/hosts-labels-and-software) for the shared model.

## Software

The software inventory shows what osquery has observed across the fleet. Titles include installed versions, paths, signing details, and host counts.

This inventory is read-only. Use [Munki](./munki) to manage desired software.

## Directory

When Entra sync is configured, users and groups appear under **Directory**. This data supports derived labels and user affinity. See [Directory](./directory).
