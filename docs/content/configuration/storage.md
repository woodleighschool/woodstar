---
sidebar_position: 2
title: Munki Storage
description: Where Munki files live and how clients get to them.
---

# Munki Storage

Munki package installers, icons, client-resources banners, and compiled client-resources archives all use Woodstar's storage abstraction. Their owning metadata stays in Postgres; the bytes live in a storage backend. A file is available only after its storage object has been uploaded and confirmed.

## Backends

`WOODSTAR_STORAGE_KIND` picks the backend:

- `file` (the default) writes bytes under `WOODSTAR_STORAGE_FILE_ROOT` on local disk. Nothing to configure; fine for a single node and for local development.
- `s3` uses an S3-compatible bucket, and adds presigned uploads and presigned download redirects.

The rest of Woodstar behaves the same either way; only the byte transfer differs.

## Getting artifacts in

Package and icon uploads are create-first. You make the Munki resource, attach a pending storage object to it, push the bytes, then confirm:

1. Create the software title or package. It can exist before it has any bytes.
2. Attach an upload. Woodstar registers a pending object and returns an upload target.
3. Upload the bytes to that target.
4. Confirm. Woodstar checks the object landed and marks it available; only then will Munki serve it.

The upload target depends on the backend. On `s3` it is a presigned `PUT` straight to the bucket, so the bytes never pass through Woodstar. On `file` it is a Woodstar URL that streams the body to disk, so large installers do not buffer in memory.

Client Resources uses the same upload path for its banner, accepting JPEG and PNG images up to 5 MiB. On **Save**, Woodstar confirms and validates the banner, builds `site_default.zip` on the server, stores the archive through the selected backend, and replaces the singleton's banner and archive references. The client never uploads a ZIP.

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
- `/munki/client_resources/*`

Each request uses the shared Munki bearer secret. Woodstar resolves the stable repository name to an available storage object, then either redirects to a presigned URL (`s3`) or streams the bytes itself (`file`). Package requests may instead redirect to an eligible distribution point; icons and client resources always use Woodstar's primary storage path. See [Munki Repository](../agent-protocols/munki-repository) for the route contracts.

## Local Garage

The checked-in compose stack runs [Garage](https://garagehq.deuxfleurs.fr/) as a local S3 backend, with path-style addressing:

```bash
WOODSTAR_STORAGE_KIND=s3
WOODSTAR_STORAGE_S3_ENDPOINT=http://garage:3900
WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT=https://garage.woodstar.test
WOODSTAR_STORAGE_S3_PATH_STYLE=true
```

The internal endpoint can stay on Garage's HTTP listener because only Woodstar uses it. Put the public endpoint behind local HTTPS and trust that development CA on clients; upload and redirect URLs are deliberately never issued over HTTP. Browser uploads still need a CORS rule on the Garage bucket. A real deployment brings its own bucket and endpoints; the full set of settings is in [Environment](./environment#storage).
