---
sidebar_position: 2
title: Munki Storage
description: Where Munki artifacts live and how clients get to them.
---

# Munki Storage

Munki package and icon artifacts are split in two. The metadata is a row in Postgres; the bytes live in a storage backend. The database always knows about an artifact, but it can only hand out the file once the bytes are uploaded and confirmed.

## Backends

`WOODSTAR_STORAGE_KIND` picks the backend:

- `file` (the default) writes bytes under `WOODSTAR_STORAGE_FILE_ROOT` on local disk. Nothing to configure; fine for a single node and for local development.
- `s3` uses an S3-compatible bucket, and adds presigned uploads and presigned download redirects.

The rest of Woodstar behaves the same either way; only the byte transfer differs.

## Getting artifacts in

Uploads are create-first. You make the Munki resource, attach a pending storage object to it, push the bytes, then confirm:

1. Create the software title or package. It can exist before it has any bytes.
2. Attach an upload. Woodstar registers a pending object and returns an upload target.
3. Upload the bytes to that target.
4. Confirm. Woodstar checks the object landed and marks it available; only then will Munki serve it.

The upload target depends on the backend. On `s3` it is a presigned `PUT` straight to the bucket, so the bytes never pass through Woodstar. On `file` it is a Woodstar URL that streams the body to disk, so large installers do not buffer in memory.

### Browser uploads to S3 need bucket CORS

The `s3` upload goes from the browser directly to the bucket, which makes it a cross-origin request. The bucket has to return CORS headers for the admin origin, or the browser blocks the `PUT`: the preflight `OPTIONS` is rejected (Cloudflare R2, for example, answers `403`) and the upload never starts. Add a CORS rule that allows the Woodstar origin with `PUT`, `GET`, and `HEAD`:

```json
[
  {
    "AllowedOrigins": ["https://woodstar.example.com"],
    "AllowedMethods": ["GET", "PUT", "HEAD"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"]
  }
]
```

This only affects browser uploads from the admin UI. AutoPkg and other server-side clients send no `Origin` and are not subject to it, and the `file` backend serves its upload URL from Woodstar's own origin, so it needs no CORS at all.

## Getting artifacts out

Munki clients never see raw storage keys. The manifests and catalogs Woodstar renders point at stable Woodstar URLs, and the client fetches through:

- `/munki/pkgs/*`
- `/munki/icons/*`

Each request is authenticated, the host is resolved by its serial, Woodstar checks the artifact actually applies to that host, and then it either redirects to a presigned URL (`s3`) or streams the bytes itself (`file`). A host only gets bytes for software it's assigned (see [Munki Repository](../agent-protocols/munki-repository)).

## Local Garage

The checked-in compose stack runs [Garage](https://garagehq.deuxfleurs.fr/) as a local S3 backend, with path-style addressing:

```bash
WOODSTAR_STORAGE_KIND=s3
WOODSTAR_STORAGE_S3_ENDPOINT=http://garage:3900
WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT=http://127.0.0.1:3900
WOODSTAR_STORAGE_S3_PATH_STYLE=true
```

That, plus the bucket, region, and credentials, is enough to exercise upload and redirect locally. Browser uploads still need a CORS rule on the Garage bucket. A real deployment brings its own bucket and endpoints; the full set of settings is in [Environment](./environment#storage).
