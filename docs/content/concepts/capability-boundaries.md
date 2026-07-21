---
sidebar_position: 1
title: How It Fits Together
description: The two sides of Woodstar, the host record everything hangs off, and how labels tie it together.
---

# How It Fits Together

Woodstar has two faces. One is the admin side: the React app you log into, backed by a JSON API. The other is the agent side: the protocol endpoints that Macs talk to. They share a database and the same host records, but they are separate surfaces with separate authentication, and it helps to keep them apart in your head.

## The admin side

This is the SPA and the API under `/api`. You sign in as a Woodstar user, and the browser carries a session cookie. Everything you do in the app (listing hosts, writing a Santa rule, scheduling an osquery report, building a Munki deployment) goes through this API. It speaks plain JSON and is documented in the [API reference](../api/overview).

## The agent side

This is where the Macs connect. Orbit, osquery, Santa, and Munki each speak their own protocol, shaped to match the existing client, so these endpoints don't look like the tidy admin API. They live on their own routes, authenticate with agent secrets and node keys rather than sessions, and are covered in [Agent Protocols](../agent-protocols/overview).

Keeping the two apart is deliberate. The admin API is the contract the frontend depends on. The agent protocols are contracts the Mac clients depend on. Neither has to bend to the other.

## Hosts are the anchor

A host is one enrolled Mac. Almost everything in Woodstar hangs off a host: its inventory, its software, its Santa state, its Munki assignments, its place in a label.

Orbit and osquery are the two clients that bring a host into existence. When a Mac enrolls, Woodstar creates or refreshes its host record. Santa and Munki don't do this. They look up a host that already exists and enrich it. If a Santa or Munki request arrives for a machine Woodstar has never seen, the protocol says so rather than inventing a host.

## Labels are how you target

A label is a group of hosts. Once you can name a group, you can point things at it: scope an osquery report, limit a check, assign a Santa configuration, decide who gets a Munki package.

Labels come in three flavours:

| Type    | Membership                                                      |
| ------- | --------------------------------------------------------------- |
| Manual  | Hosts you pick by hand.                                         |
| Dynamic | Hosts that match an osquery query.                              |
| Derived | Hosts that match directory data, such as a department or group. |

Dynamic labels depend on osquery results coming back from the fleet. Derived labels depend on directory sync. Manual labels just are what you set them to.

## The four capabilities

Each agent has an admin area to match it:

- **Hosts and inventory** is the observed state: who's enrolled, what hardware, what software, what users.
- **osquery** is reports, checks, and live queries against the fleet.
- **Santa** is execution policy: configurations, rules, and the events that come back.
- **Munki** is desired software state: titles, packages, and deployments.

The pages under [Admin Guide](../admin/hosts-and-inventory) walk through each one.
