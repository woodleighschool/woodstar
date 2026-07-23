---
sidebar_position: 4
title: Munki
description: Managed software, packages, client resources, and distribution points.
---

# Munki

Woodstar stores Munki software and package metadata, builds manifests and catalogs for each Mac, and serves package files and Managed Software Center resources.

## Software

A software title represents one managed item, such as Google Chrome. Each title contains a name, description, category, developer, icon, and targets. Package versions belong to the title.

## Packages

A package contains the Munki metadata for one version. Woodstar supports `pkg`, `copy_from_dmg`, and `nopkg` installer types.

`pkg` and `copy_from_dmg` packages require an uploaded installer. A `nopkg` item has no installer file. The package form includes Munki settings such as restart behaviour, supported architectures, blocking applications, requirements, update relationships, install checks, and uninstall methods.

Most packages are created through [AutoPkg](../autopkg/overview), but they can also be entered or imported in the web app.

## Targets

Targets add a software title to Munki manifest sections for matching labels.

| Action            | Result                                     |
| ----------------- | ------------------------------------------ |
| Managed install   | Keep the item installed                    |
| Managed uninstall | Keep the item removed                      |
| Managed update    | Keep an installed title up to date         |
| Optional install  | Offer the title in Managed Software Center |
| Featured item     | Highlight an optional install              |
| Default install   | Select an optional install by default      |

Each include chooses the latest package or a specific version. Includes are evaluated in order, and the first matching label wins. Any matching exclude removes the title from the host's manifest.

## Client Resources

Client Resources changes the appearance and links in Managed Software Center. Use the builder to add a JPEG or PNG banner, navigation links, and a footer, or upload a complete resources ZIP.

The banner limit is 5 MiB. **Fit** keeps the image at the preview height; **Fill** crops the image to the banner area. Links can use `http`, `https`, `mailto`, or `munki` targets.

Changes are published when you select **Save**. **Cancel** restores the saved values. **Undeploy** stops serving the archive, so clients return to Munki's built-in resources.

## Distribution points

A distribution point is a local cache for package installers. Woodstar continues to provide manifests and catalogs, while matching Munki clients download installers directly from the cache.

Each point has:

- An HTTPS URL that its Munki clients can resolve and reach.
- Client source CIDRs that match the address Woodstar derives from each package request.
- A position used when more than one point matches.
- A key shown when the point is created or rotated.

A point with no client CIDRs receives no redirects. Use `0.0.0.0/0` and `::/0` for a catch-all. See [how client matching works](../agent-protocols/munki-distribution#how-client-matching-works) before assigning ranges behind a proxy.

The package list shows whether each installer is pending, syncing, current, or in error. Woodstar selects a point only while the worker is online and the requested package is current. Otherwise, the package comes from Woodstar's primary storage.

Woodstar cannot check the route from a Mac to the cache. If a client's address matches a point but DNS, routing, firewall rules, or TLS prevent access to its URL, the Munki download fails. Woodstar has already redirected the request and cannot serve that attempt from primary storage.

See [Munki Distribution Points](../agent-protocols/munki-distribution) to run the worker.
