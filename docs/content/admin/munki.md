---
sidebar_position: 4
title: Munki
description: "Desired software state, client presentation, and repository delivery."
---

# Munki

Munki is where you decide what software a Mac should have and how Managed Software Center presents it. Woodstar holds the managed catalog, publishes client resources, and renders the repository data that Munki clients pull down. This is desired state, the opposite end from the [observed software inventory](./hosts-and-inventory#software) that osquery reports.

The Munki admin surface has five parts.

## Software titles

A title is the human-owned grouping for one managed thing, "Google Chrome", say. It carries the name, description, category, developer, and icon. The icon can be a fresh upload or one picked from those already in use by other titles. Everything else hangs off a title.

## Packages

A package is one complete installable version under a title. Packages keep Munki's own vocabulary because that is what ends up in the rendered pkginfo: installer type, restart action, blocking applications, `requires`, `update_for`, supported architectures, and the unattended flags. `pkg` and `copy_from_dmg` require finalized installer bytes; only `nopkg` is packageless. There is no separate availability control.

You can create package metadata by hand, or import an existing pkginfo item. In practice most packages arrive through [AutoPkg](../autopkg/overview) rather than being typed in.

## Targets

Targets are how a title reaches machines. Each include selects a label, chooses the latest package or pins a specific version, and sets one or more Munki manifest actions:

- Managed installs keep the item installed.
- Managed uninstalls keep the item removed.
- Managed updates update the item only where some version is already installed.
- Optional installs make the item available in Managed Software Center.
- Featured items highlight an optional install.
- Default installs initially select an optional install for the user.

Optional installs and managed updates can be used together. The item remains available for a user to install, while any installed copy must take an available update. Managed updates alone never install an absent item.

Includes are ordered from highest to lowest priority. When a host matches more than one include label for the same title, the first one wins. A matching exclude label suppresses the title regardless of which include matched.

## Artifacts

An artifact is the actual file: the package payload or an icon. The metadata is a stable Woodstar row; the bytes live in a storage backend, local files by default or an S3-compatible bucket.

For an installer-backed package, Woodstar reserves an unclaimed installer object, uploads and finalizes it, then creates or replaces the package with that object. A failed package mutation removes the new unclaimed object and leaves an existing package untouched. Storage always works (the default `file` backend needs no setup); S3 adds direct and multipart bucket uploads. See [Munki Storage](../configuration/storage) for the transfer details, and [Munki Repository](../agent-protocols/munki-repository) for how clients fetch it.

## Client resources

Client Resources controls the single Managed Software Center resources archive served by Woodstar. Until you save a configuration, clients use Munki's built-in resources.

You can build the archive in Woodstar or upload a ZIP you maintain yourself.

The builder has:

- One JPEG or PNG banner, up to 5 MiB. **Fit** preserves the image at the preview height, while **Fill** crops it to cover the banner area. **Left** and **Centre** control the horizontal focal position.
- Optional links above the software list. When the list is empty, Managed Software Center shows its normal categories. Adding a link replaces that category list.
- Optional footer text and links. An empty footer is not included in the archive.

Links keep their displayed order and can use `http`, `https`, `mailto`, or `munki` targets. Only HTTP and HTTPS links can open in the system browser.

The builder is also the preview. It follows the Managed Software Center layout closely, but the native app remains the rendering authority.

**Upload Custom ZIP** switches the form to an archive attachment. Selecting a file does not upload or publish it; **Save** applies whichever form is visible. For the builder, Woodstar validates the banner and compiles the archive. For a custom ZIP, Woodstar stores and publishes the supplied archive as-is. Switching forms does not discard either form's pending values, and publishing a custom ZIP keeps the saved builder configuration available for switching back later.

**Cancel** restores the last saved values. **Undeploy** removes the configured resources and returns clients to Munki's built-in resources.

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
