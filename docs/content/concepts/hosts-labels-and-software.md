---
sidebar_position: 2
title: Hosts, Labels, and Software
description: The three objects most admin views are built on.
---

# Hosts, Labels, and Software

Most of the admin app starts from one of three objects: a host, a label, or a software title. Santa and Munki add their own views, but those still resolve back to hosts and labels.

## Hosts

A host is an enrolled Mac. The host record carries its identity, hardware, OS, network, storage, agent versions, enrollment metadata, the user it's associated with, and timestamps.

Those fields are filled by osquery's detail queries, the standard set Woodstar runs on every host on a schedule without you authoring anything. This is Fleet's host-vitals model: the host record is a projection of what those queries last reported.

Open a host and you get the detail view, which pulls in everything attached to it:

- labels it belongs to
- local user accounts
- batteries
- certificates
- observed software
- osquery report and check results
- Santa state, if Santa is in play
- Munki state, if Munki is in play

The admin UI shows a single `display_name` for each host. The raw hostname, computer name, serial, and UUID are all still there as data; there just isn't a cascade of alternate names to reason about.

## Labels

Labels group hosts so you can target them. There are three membership types:

| Type | Meaning |
| --- | --- |
| `manual` | Hosts selected by hand. |
| `dynamic` | Membership from an osquery query. |
| `derived` | Membership from directory attributes such as department, group, or user. |

Manual, dynamic, and builtin labels come straight from Fleet. Builtins are the platform groupings like All Hosts and macOS; dynamic labels rely on osquery distributed query results; manual labels are whatever you set. Derived is Woodstar's addition, sourced from directory sync rather than from the fleet itself.

## Software

Software is *observed* inventory. It's populated from osquery and records titles, versions, install paths, browser extensions, bundle identifiers, signing data, and how many hosts carry each one.

This is not the same thing as Munki. Munki is desired state: the software you've decided a Mac should have. osquery seeing a package installed is an observation, and it lives in software inventory. The two sit side by side and answer different questions: "what's on this Mac?" versus "what should be?"

The observed side is Fleet's software inventory model. The managed side, which Fleet handles with its own software library, is Munki's job in Woodstar.

## User affinity

User affinity is Woodstar's best guess at which person a Mac belongs to. It's stitched together from a few sources: Orbit can report an email from the device's profile, Santa can report the primary user it sees, and directory sync can flesh that user out with a name, department, and groups.

It's useful for filtering and for the host detail view. It is not an authentication source, and nothing logs in as a result of it.
