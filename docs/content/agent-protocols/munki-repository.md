---
sidebar_position: 4
title: Munki Repository
description: Munki client repository routes and artifact redirect behavior.
---

# Munki Repository

Woodstar exposes a Munki repository-shaped HTTP surface under `/munki`. It renders manifests and catalogs from Woodstar-managed software, package, deployment, and artifact rows.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/munki/manifests/{name}` | Render a manifest plist for the requesting host. |
| `GET` | `/munki/catalogs/{name}` | Render a catalog plist. The current code accepts `production`. |
| `GET` | `/munki/pkgs/*` | Redirect to a package artifact if the requesting host can fetch it. |
| `GET` | `/munki/icons/*` | Redirect to an icon artifact. |

## Request Identity

Munki routes require:

```http
Authorization: Bearer <munki-agent-secret>
Serial: <mac-hardware-serial>
```

The service resolves `Serial` to an existing host by hardware serial. Unknown serials return `404`.

## Manifests And Catalogs

Manifest rendering starts from effective packages for the host. Deployments can place package names into:

- `managed_installs`
- `managed_uninstalls`
- `managed_updates`
- `optional_installs`
- `featured_items`

Catalog rendering builds pkginfo items from the effective package set. Package selection can follow the latest eligible package or a pinned package.

## Artifacts

Artifacts are stable Woodstar rows. Package and icon artifact routes redirect to object storage when the Munki S3 presigner is configured.

If the artifact exists but no storage backend is usable, the protocol route returns `503 Service Unavailable`. If a package artifact is not part of the host's effective package set, the route returns `404`.
