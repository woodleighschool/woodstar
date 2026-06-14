---
sidebar_position: 4
title: Munki Repository
description: "The Munki repository surface: manifests, catalogs, and artifact redirects."
---

# Munki Repository

To a Munki client, Woodstar looks like a Munki repository. It serves the same shape of URLs a static repo would, but the manifests and catalogs are rendered on the fly from the software, packages, deployments, and artifacts you've set up.

| Method | Path                      | Purpose                                                             |
| ------ | ------------------------- | ------------------------------------------------------------------- |
| `GET`  | `/munki/manifests/{name}` | Render the manifest plist for the requesting host.                  |
| `GET`  | `/munki/catalogs/{name}`  | Render a catalog plist. The `production` catalog is the one in use. |
| `GET`  | `/munki/pkgs/*`           | Redirect to a package artifact, if this host is allowed it.         |
| `GET`  | `/munki/icons/*`          | Redirect to an icon artifact.                                       |

## Request identity

Every request carries the shared secret and the machine's serial:

```http
Authorization: Bearer <munki-agent-secret>
Serial: <mac-hardware-serial>
```

Woodstar resolves the `Serial` to an existing host. An unknown serial gets a `404`; there's no host to render a manifest for.

## How a manifest comes together

Rendering starts from the host's effective package set, which is whatever the applicable deployments work out to. Each package lands in the Munki key its deployment chose:

- `managed_installs`
- `managed_uninstalls`
- `managed_updates`
- `optional_installs`
- `featured_items`

The catalog is built from the same effective set. When more than one version of a package is eligible, selection follows either the latest one or a version pinned in the deployment.

## Artifacts

Artifacts are stable Woodstar rows; the package and icon routes hand back the bytes by redirecting to a presigned URL on the S3 backend, or streaming them directly on the file backend.

The failure case worth knowing: if a package artifact isn't part of this host's effective set, the route returns `404`, even if the file is sitting in storage. A Mac only gets the bytes for software it's actually assigned. The storage wiring is in [Munki Storage](../configuration/storage).
