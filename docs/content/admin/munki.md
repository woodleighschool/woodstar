---
sidebar_position: 4
title: Munki
description: "Desired software state: titles, packages, deployments, and artifacts."
---

# Munki

Munki is where you decide what software a Mac should have. Woodstar holds the managed catalog and renders the manifests and catalogs that Munki clients pull down. This is desired state, the opposite end from the [observed software inventory](./hosts-and-inventory#software) that osquery reports.

Four objects make it work.

## Software titles

A title is the human-owned grouping for one managed thing, "Google Chrome", say. It carries the name, description, category, developer, and icon. The icon can be a fresh upload or one picked from those already in use by other titles. Everything else hangs off a title.

## Packages

A package is one installable version under a title. Packages keep a lot of Munki's own vocabulary, because that's what ends up in the rendered pkginfo: installer type, restart action, blocking applications, `requires`, `update_for`, supported architectures, the unattended flags, and an `extra_pkginfo` escape hatch for anything Woodstar doesn't model directly.

You can create package metadata by hand, or import an existing pkginfo item. In practice most packages arrive through [AutoPkg](../autopkg/overview) rather than being typed in.

## Deployments

A deployment is how a title reaches machines. It targets all hosts, or includes and excludes labels, or includes and excludes specific host IDs, and it carries an action:

- `install`
- `remove`
- `update_if_present`
- `none`

Deployments are ordered, because a host can match more than one and the order decides which wins. When Woodstar renders a manifest for a Mac, it works out the effective set of packages from the deployments that apply and drops each one into the right Munki key (managed installs, managed uninstalls, managed updates, optional installs, or featured items).

## Artifacts

An artifact is the actual file: the package payload or an icon. The metadata is a stable Woodstar row; the bytes live in a storage backend, local files by default or an S3-compatible bucket.

An artifact gets in create-first: the title or package exists first, then you attach an upload, push the bytes, and confirm them. Storage always works (the default `file` backend needs no setup), so the only real choice is whether bytes stream through Woodstar or go straight to a bucket. See [Munki Storage](../configuration/storage) for how the backends differ, and [Munki Repository](../agent-protocols/munki-repository) for how clients fetch it.

## Distribution points

A distribution point is a mirror node you run near a group of Macs so package installers download from the local network instead of from Woodstar or its bucket every time. Woodstar still owns what's installed and holds the canonical files; the distribution point just keeps copies and serves them. This is the admin side: creating the points and watching their state. How the worker mirrors and serves is in [Munki Distribution Points](../agent-protocols/munki-distribution).

A point carries a name, an enabled flag, the client CIDRs whose Macs it serves, and the public base URL Macs reach it on. The CIDRs are exact: a point with no CIDRs serves nobody, and a catch-all has to be spelled out as `0.0.0.0/0` and `::/0`. Points are ordered, and when a Mac matches more than one, the first eligible point in the order wins, so the order is your preference list. The same "Edit order" reordering you use for Santa configurations applies here.

Each point has its own key, shown once when you create the point and once when you rotate it. The worker authenticates with it and Woodstar signs download grants with it, so treat it like any other secret. If a key leaks or a worker is decommissioned, rotate to cut the old one off.

The detail view lists every package and its state on that point:

- `pending` -- desired, but the point hasn't reported it yet.
- `syncing` -- reported, but not yet the current installer; the point is still catching up.
- `current` -- mirrored and matching the installer Woodstar would serve.
- `error` -- the point couldn't mirror it; the reported reason is shown.

A Mac is only redirected to a point for packages it has `current`, and only while the point is online, so a point that's mid-sync or disconnected quietly falls back to Woodstar-direct delivery.

Endpoints are in the [API reference](../api/overview).
