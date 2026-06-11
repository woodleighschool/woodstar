---
sidebar_position: 1
title: Introduction
description: What Woodstar is, why it exists, and how the pieces fit.
---

# Woodstar

Woodstar is a self-hosted server for managing a fleet of Macs. It keeps an inventory of every enrolled machine, decides what software those machines should run and which binaries are allowed to launch, and serves the agents on each Mac.

We built it at Woodleigh to fill the gaps left when we moved Mac management from Jamf to Intune.

## Why it exists

Intune is the MDM now. It handles enrollment, configuration profiles, and the rest of what an MDM is good at. What it doesn't do well is the Mac-specific ground Jamf used to cover, so Woodstar picks that up:

- **Software and patching** run through Munki. Woodstar is the Munki server. It holds the catalog of managed software and works out which packages each Mac should install, update, or remove.
- **What's allowed to run** is Santa's job. Woodstar is the Santa sync server. It ships allow and block rules to each Mac and collects the execution events Santa sends back.
- **Inventory and custom facts** come from osquery. This is the stand-in for Jamf's Extension Attributes and Smart Groups: ask the fleet a SQL question, save it as a scheduled report or a pass/fail check, and group hosts with labels.

So Intune owns the device. Woodstar owns the software, the execution policy, and the inventory Intune doesn't cover.

## The agents

A Mac talks to Woodstar through four clients. Two of them get the machine on board and keep its record current; the other two attach to a machine that already exists.

| Agent | Does |
| --- | --- |
| Orbit | Enrolls the Mac and reports back the basics. Compatible with the Fleet Orbit client. |
| osquery | Runs the queries behind inventory, reports, checks, and live queries. |
| Santa | Enforces which binaries can run and reports execution events. |
| Munki | Installs, updates, and removes managed software. |

Hosts are the thing everything hangs off. Orbit and osquery create and refresh hosts during enrollment; Santa and Munki look up a host that's already there. Labels are how you group hosts and aim everything else at them.

## Built on Fleet's core

The host, label, query, and check model underneath all this is [Fleet's](https://fleetdm.com/). Woodstar takes Fleet's osquery core and its reporting, then builds the macOS-specific pieces around it with Santa and Munki. It isn't all of Fleet: there's no MDM here, the platform focus is macOS, and the surface is trimmed to what our fleet needs. Where you see hosts, labels, scheduled reports, checks, and Orbit enrollment, you're looking at Fleet's model. Santa and Munki are the parts we added.

## How these docs are organized

- [Getting Started](./getting-started/local-development) gets Woodstar running from a checkout or the compose stack.
- [Concepts](./concepts/capability-boundaries) is the mental model: the two sides of the server, hosts, labels, and agent secrets.
- [Admin Guide](./admin/hosts-and-inventory) covers the day to day in the admin app.
- [Agent Protocols](./agent-protocols/overview) is how the four clients enroll, authenticate, and sync.
- [AutoPkg](./autopkg/overview) is how packages get authored and pushed into Woodstar's Munki repo.
- [Configuration](./configuration/environment) and the [API reference](./api/overview) are the lookup material.
