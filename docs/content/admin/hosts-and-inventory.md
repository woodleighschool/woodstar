---
sidebar_position: 1
title: Hosts and Inventory
description: Hosts, labels, software inventory, and directory data.
---

# Hosts and Inventory

The host list shows every enrolled Mac. Search and filters can narrow the list by status, label, software, or osquery policy result.

## Host details

A host page combines its hardware and operating-system details with accounts, batteries, certificates, software, labels, and osquery results. Agent-specific tabs appear after the corresponding agent contacts the server. Empty tables retain their columns and controls. The host header shows when inventory was fetched.

Use **Assigned User** to correct or clear the person associated with a Mac. Manual assignment takes precedence over the Orbit profile. Logged-in accounts do not change assignment or derived labels.

### Accounts

The **Accounts** tab shows usernames, names, administrator group membership, Secure Token status, and FileVault unlock eligibility. A green dot beside a username means the account was logged in at the last inventory. Expand a row for its UID, UUID, home directory, shell, hidden status, and volume ownership.

**Login Accounts** includes logged-in accounts and accounts with an interactive shell, administrator membership, or FileVault access. **Other Accounts** contains the remaining accounts in a separate table. Hidden accounts can appear in either group.

Accounts and logged-in state are collected together with the hourly inventory and when **Refresh Inventory** is requested. Each successful account collection replaces the previous snapshot.

Security values are nullable when the relevant extension is unavailable, collection fails, or the account has no UUID. FileVault eligibility comes from `filevault_users`. Secure Token and volume ownership come from `apfs_crypto_users`, scoped to the mounted startup Data volume using `apfs_volumes` and `mounts`. Both remain null without crypto-user rows from that volume. These extension tables depend on the installed fleetd version.

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
