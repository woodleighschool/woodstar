---
sidebar_position: 1
title: Hosts and Inventory
description: Hosts, labels, software inventory, and directory data.
---

# Hosts and Inventory

The host list shows every enrolled Mac. Search and filters can narrow the list by status, label, software, or osquery check result.

## Host details

A host page combines its hardware and operating-system details with users, batteries, certificates, software, labels, and osquery results. Santa and Munki sections appear after those clients have reported state.

Use **User affinity** to correct or clear the person associated with a Mac.

## Labels

Labels group hosts for reports, checks, Santa policy, and Munki software.

- **Manual** labels contain selected hosts.
- **Dynamic** labels use an osquery query.
- **Derived** labels use directory users, groups, or departments.

See [Hosts, Labels, and Software](../concepts/hosts-labels-and-software) for the shared model.

## Software

The software inventory shows what osquery has observed across the fleet. Titles include installed versions, paths, signing details, and host counts.

This inventory is read-only. Use [Munki](./munki) to manage desired software.

## Directory

When Entra sync is configured, Woodstar lists its users and groups under **Directory**. This data supports derived labels and user affinity. See [Directory](./directory).
