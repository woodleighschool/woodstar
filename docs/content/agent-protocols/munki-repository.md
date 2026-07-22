---
sidebar_position: 4
title: Munki Repository
description: Configure Munki to use Woodstar as its repository.
---

# Munki Repository

Woodstar serves manifests, catalogs, packages, icons, and Managed Software Center resources using Munki's repository layout.

## Configure Munki

Create a Munki secret under **Enrollments > Munki**, then copy the generated configuration profile. The profile sets:

- `SoftwareRepoURL` to `<WOODSTAR_URL>/munki`
- `ClientIdentifier` to the MDM-expanded serial number
- `FollowHTTPRedirects` to `https`
- The Munki secret in `AdditionalHttpHeaders`

The serial number must belong to an enrolled Woodstar host.

## Routes

| Method | Path                                       | Purpose                                    |
| ------ | ------------------------------------------ | ------------------------------------------ |
| `GET`  | `/munki/manifests/{serial}`                | Manifest for one host                      |
| `GET`  | `/munki/catalogs/woodstar`                 | Shared package catalog                     |
| `GET`  | `/munki/pkgs/*`                            | Package installer                          |
| `GET`  | `/munki/icons/*`                           | Software icon                              |
| `GET`  | `/munki/client_resources/{serial}.zip`     | Managed Software Center resources          |
| `GET`  | `/munki/client_resources/site_default.zip` | Managed Software Center fallback resources |

Every request uses the Munki bearer secret. An unknown manifest serial returns `404`.

## Manifests and catalogs

Woodstar builds each manifest from the host's matching software targets. A latest-version target uses the bare Munki name; a pinned package uses `name--version`.

The `woodstar` catalog contains every package. Installer-backed packages include their location, size, and SHA-256. `nopkg` items omit the installer fields.

## Files

With file storage, Woodstar streams artifacts. With S3, Woodstar redirects to a presigned URL. Matching package requests may be redirected to a [distribution-point cache](./munki-distribution); icons and client resources always use primary storage.

When no Client Resources archive is deployed, that route returns `404` and Munki uses its built-in resources.
