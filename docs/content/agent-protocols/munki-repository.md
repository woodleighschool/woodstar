---
sidebar_position: 4
title: Munki Repository
description: "The Munki repository surface: manifests, catalogs, artifacts, and client resources."
---

# Munki Repository

To a Munki client, Woodstar looks like a Munki repository. It serves the same shape of URLs a static repo would, but manifests, catalogs, and client resources are rendered or compiled from the state managed in Woodstar.

| Method | Path                                       | Purpose                                                           |
| ------ | ------------------------------------------ | ----------------------------------------------------------------- |
| `GET`  | `/munki/manifests/{serial}`                | Render the manifest plist for the host with that hardware serial. |
| `GET`  | `/munki/catalogs/woodstar`                 | Render the shared Woodstar catalog plist.                         |
| `GET`  | `/munki/pkgs/*`                            | Deliver a package artifact.                                       |
| `GET`  | `/munki/icons/*`                           | Deliver an icon artifact.                                         |
| `GET`  | `/munki/client_resources/{serial}.zip`     | Deliver the configured archive for a known host.                  |
| `GET`  | `/munki/client_resources/site_default.zip` | Deliver the configured fallback archive.                          |

## Request identity

Every repository request carries the shared Munki agent secret:

```http
Authorization: Bearer <munki-agent-secret>
```

The generated profile sets `ClientIdentifier` to the MDM-expanded serial number, so Munki requests `/munki/manifests/{serial}`. Woodstar resolves that URL name to an existing host before rendering the manifest. An authenticated request for an unknown serial, `site_default`, or any other non-host manifest name gets `404`; Woodstar does not synthesize fallback manifests.

Catalog, package, icon, and `site_default.zip` routes use the bearer secret but do not resolve a host. A client-resources request named for a serial must resolve to an existing host. Both accepted client-resources names serve the same configured singleton archive.

## How a manifest comes together

Manifest rendering starts from the host's effective package set, which is whatever the applicable targets work out to. Each package lands in every Munki key selected by its effective include:

- `managed_installs`
- `managed_uninstalls`
- `managed_updates`
- `optional_installs`
- `default_installs`
- `featured_items`

The manifest always references the `woodstar` catalog. When an include asks for the latest release, the manifest references the bare Munki name; when it pins a release, the manifest references Munki's `name--version` syntax.

The catalog is shared across hosts and contains every persisted package. `pkg` and `copy_from_dmg` packages always have a finalized installer, so their pkginfo always carries `installer_item_location`, `installer_item_size`, and the whole-file SHA-256 in `installer_item_hash`. `nopkg` omits those fields. There is no package eligibility switch or installer-availability filter. If stored package and object state ever break that invariant, catalog rendering fails instead of emitting an incomplete pkginfo.

## Artifacts

Installer objects are reserved, uploaded, and finalized before the package is created or replaced. Each finalized installer has exactly one package owner. Icons retain their resource-scoped upload lifecycle. The package and icon routes authorize the bearer secret, resolve the requested `installer_item_location` or icon name against the shared repository objects, and then hand back the bytes by redirecting to a presigned URL on the S3 backend or streaming them directly on the file backend.

If the path does not match a known package installer or icon in the repository, the route returns `404`. The storage wiring is in [Munki Storage](../configuration/storage).

## Client resources

Saving [Client Resources](../admin/munki#client-resources) compiles the selected banner, optional links, and optional footer into a stored `site_default.zip`. Munki asks for the host-specific `{serial}.zip` first and can fall back to `site_default.zip`; Woodstar serves the same configured archive for either accepted name.

The route uses the shared Munki bearer secret and delivers the archive through the same storage path as icons and packages: a presigned redirect on S3, or a Woodstar stream with file storage. Client resources are not replicated to distribution points.

When no client resources are configured, or the stored archive is unavailable, the route returns `404` and Munki uses its built-in resources.

## Distribution points

The `/munki/pkgs/*` redirect doesn't always point at storage. If a [distribution point](./munki-distribution) is mirroring this package and covers the requesting Mac's IP, Woodstar redirects there instead, so the download comes off the local network. The Mac follows the redirect the same way it would for any other; it doesn't know or care which one it got.

This only applies to package installers, the heavy downloads. Icons and client resources are always served directly by Woodstar. And it's best-effort: if no distribution point is eligible, or Woodstar can't work out the client's IP, the package falls back to direct storage delivery.
