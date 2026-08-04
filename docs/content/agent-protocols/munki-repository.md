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
- `FollowHTTPRedirects` to `https`
- `Authorization: Bearer <MUNKI_SECRET>` in `AdditionalHttpHeaders`
- `X-Woodstar-Serial-Number: <MDM_EXPANDED_SERIAL>` in `AdditionalHttpHeaders`

Both headers accompany every repository request. The bearer secret authenticates the Munki
client and the serial header binds the request to an enrolled Woodstar host. Munki does not
need a `ClientIdentifier`, UUID field, or preflight/postflight hook for Woodstar.

## Routes

| Method | Path                              | Purpose                           |
| ------ | --------------------------------- | --------------------------------- |
| `GET`  | `/munki/manifests/{name}`         | Host-scoped manifest              |
| `GET`  | `/munki/catalogs/woodstar`        | Host-scoped package catalog       |
| `GET`  | `/munki/icons/_icon_hashes.plist` | Host-scoped icon hash list        |
| `GET`  | `/munki/pkgs/*`                   | Package installer                 |
| `GET`  | `/munki/icons/*`                  | Software icon                     |
| `GET`  | `/munki/client_resources/*`       | Managed Software Center resources |

Path components select a Munki repository object; they do not identify the requesting Mac.
For example, `manifests/{name}` is a manifest selector and a package path selects an installer.
Woodstar always resolves the requester from `X-Woodstar-Serial-Number` before serving the
selected object.

A missing or invalid bearer secret returns `401`. A missing or unknown serial, an unavailable
repository object, or an object not assigned to that host returns `404`; this deliberately
blackholes objects outside the host's repository view.

## Manifests and catalogs

Woodstar builds each manifest and `woodstar` catalog from the host's matching software targets.
A latest-version target uses the bare Munki name; a pinned package uses `name--version`.

Installer-backed catalog items include their location, size, and SHA-256. `nopkg` items omit
the installer fields. Package and icon files are available only when the selected item belongs
to the requesting host's catalog.

## Files

With file storage, Woodstar streams artifacts. With S3, Woodstar redirects to a presigned URL. Matching package requests may be redirected to a [distribution-point cache](./munki-distribution); icons and client resources always use primary storage.

Client Resources is one shared archive, but Woodstar still resolves and authorizes the host
before it serves the archive. Munki may request any Client Resources path; the path is only a
repository selector. When no archive is deployed, the route returns `404` and Munki uses its
built-in resources.
