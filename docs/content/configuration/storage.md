---
sidebar_position: 2
title: Munki Storage
description: Where Munki artifacts live and how clients get to them.
---

# Munki Storage

Munki package and icon artifacts are split in two. The metadata is a row in Postgres; the actual bytes live in S3-compatible object storage. The database always knows about the artifact, but it can only serve the file when storage is configured.

## Getting artifacts in

There are two ways an artifact arrives, both from the admin side:

1. Register metadata for a file that already sits at a known storage location.
2. Ask Woodstar for a temporary upload URL, push the file straight to storage, then attach it to a package.

The upload-URL route needs a storage presigner, so it only works once S3 is set up.

## Getting artifacts out

Munki clients never see raw storage keys. The manifests and catalogs Woodstar renders point at stable Woodstar URLs, and the client fetches through:

- `/munki/pkgs/*`
- `/munki/icons/*`

Each request is authenticated, the host is resolved by its serial, Woodstar checks the artifact actually applies to that host, and then it redirects to a presigned storage URL. A host only gets bytes for software it's assigned (see [Munki Repository](../agent-protocols/munki-repository)).

## Local Garage

The checked-in compose stack runs [Garage](https://garagehq.deuxfleurs.fr/) as a local S3 backend, with path-style addressing:

```bash
WOODSTAR_MUNKI_S3_ENDPOINT=http://garage:3900
WOODSTAR_MUNKI_S3_PUBLIC_ENDPOINT=http://127.0.0.1:3900
WOODSTAR_MUNKI_S3_PATH_STYLE=true
```

That's enough to exercise upload and redirect locally. A real deployment brings its own bucket, credentials, and endpoints; the full set of settings is in [Environment](./environment#munki-s3-storage).
