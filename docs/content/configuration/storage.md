---
sidebar_position: 2
title: Munki Storage
description: How Munki artifact storage and redirects are wired.
---

# Munki Storage

Munki package and icon artifacts are represented by database rows. The binary object lives in an S3-compatible storage backend when that backend is configured.

## Admin Flow

The admin API supports two artifact paths:

1. Register metadata for an artifact that already has a stable `location`.
2. Ask Woodstar for a temporary upload URL, upload to object storage, then attach the artifact to a package.

The upload URL path requires a configured storage presigner.

## Client Flow

Munki clients do not receive raw object-storage keys in pkginfo. Woodstar renders stable artifact URLs in manifest/catalog output and handles client requests through:

- `/munki/pkgs/*`
- `/munki/icons/*`

Those routes authorize the client, resolve its host by serial, check whether the package artifact applies to the host, and redirect to a presigned object-storage URL.

## Local Garage

The local compose stack uses Garage with path-style S3 access:

```bash
WOODSTAR_MUNKI_S3_ENDPOINT=http://garage:3900
WOODSTAR_MUNKI_S3_PUBLIC_ENDPOINT=http://127.0.0.1:3900
WOODSTAR_MUNKI_S3_PATH_STYLE=true
```

That is enough for local upload and redirect testing. Real deployments need storage credentials and endpoint behavior chosen by the operator.
