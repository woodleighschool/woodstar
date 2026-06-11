---
sidebar_position: 1
title: Hosts and Inventory
description: The host list, host detail, software inventory, labels, and directory data.
---

# Hosts and Inventory

Hosts are the main thing you work with. The host list is the front page of the fleet: every enrolled Mac, with search, status, label, software, and osquery check-result filters on top so you can narrow down to the machines you care about.

## Host detail

Open a host and Woodstar assembles the detail view from the base record plus everything attached to it: labels, local users, batteries, certificates, observed software, osquery report and check state, and (when those agents are in play) Santa and Munki sections.

The Santa and Munki parts are contributed by those capabilities rather than baked into the host itself, so a deployment without Santa or Munki simply doesn't show those sections.

From here you can also set or clear a manual **user affinity** mapping when the automatic guess is wrong or missing.

## Labels

Labels group hosts so you can target the rest of the app at them. You create them here and they show up everywhere that takes a scope: osquery reports and checks, Santa configurations and rules, Munki deployments.

- **Manual** labels hold a list of host IDs you pick.
- **Dynamic** labels hold an osquery query, and membership is whatever matches.
- **Derived** labels hold criteria against directory data, such as a department, group, or user.

See [Hosts, Labels, and Software](../concepts/hosts-labels-and-software) for how the three types behave.

## Software

The software views are observed inventory, projected from what osquery reports across the fleet. A software title rolls up its versions, where it's installed, signing details, and a host count, so you can answer "who has this, and which version?".

This is read-only and it is not Munki. It's what's actually on the Macs, not what you've decided should be. For desired state, see [Munki](./munki).

## Directory data

If directory sync is configured, the synced users, groups, and departments show up here too, and they feed derived labels and user affinity. Sync only runs when the Entra tenant, client ID, and secret are all set; see [Directory](./directory).

The exact request and response shapes for all of the above are in the [API reference](../api/overview).
