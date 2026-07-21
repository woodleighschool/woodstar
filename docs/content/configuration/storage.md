---
sidebar_position: 2
title: Munki Storage
description: Where Munki files live and how clients get to them.
---

# Munki Storage

Munki package installers, icons, client-resources banners, and client-resources archives all use Woodstar's storage backends. Their owning metadata stays in Postgres; the bytes live on disk or in a bucket. A file is available only after its storage object has been uploaded and finalized.

## Backends

`WOODSTAR_STORAGE_KIND` picks the backend:

- `file` (the default) writes bytes under `WOODSTAR_STORAGE_FILE_ROOT` on local disk and requires one capability key; fine for a single node and for local development.
- `s3` uses an S3-compatible bucket, and adds presigned uploads and presigned download redirects.

The rest of Woodstar behaves the same either way; only the byte transfer differs.

The file backend issues short-lived HMAC capabilities for Woodstar's `/storage/*` transfer routes. Set `WOODSTAR_STORAGE_CAPABILITY_KEY` to 32 random bytes encoded as hexadecimal (`openssl rand -hex 32`). S3 has no Woodstar transfer route and does not use this key; its provider signs the presigned URLs. `WOODSTAR_STORAGE_TRANSFER_TTL` controls both kinds of short-lived transfer URL.

## Getting package installers in

Installer-backed packages use one installer-first lifecycle:

1. Reserve an unclaimed package-installer object.
2. Upload its bytes.
3. Finalize it. Woodstar derives the content type, size, and whole-file SHA-256 and marks the object available.
4. Create or fully replace the package with that `installer_object_id`.

A `pkg` or `copy_from_dmg` package cannot be persisted without its finalized installer. A `nopkg` package has no installer object. Packages are therefore available to catalogs and targeting as soon as they exist; there is no separate eligibility or availability switch.

On `file`, upload is one raw `PUT` to a Woodstar URL. The file backend has no multipart operations. On `s3`, files up to 100 MiB use the existing presigned single `PUT`; larger browser uploads use S3 multipart upload and presigned part `PUT`s. Multipart completion assembles bytes at the immutable canonical object key, then normal finalization streams that object once through Woodstar to calculate Munki's whole-file SHA-256. A multipart ETag is never used as `installer_item_hash`.

Canceling an upload aborts an open S3 multipart upload and removes the unclaimed object. Configure the bucket's incomplete-multipart lifecycle rule for abandoned uploads that never reach explicit cancellation.

Icons keep their resource-scoped reserve, upload, and attach lifecycle. The Client Resources builder uses the same scoped upload path for its banner, accepting JPEG and PNG images up to 5 MiB. On **Save**, Woodstar finalizes and validates the banner, builds the archive on the server, and publishes it through the selected backend.

A custom ZIP uses a separate client-resources archive upload. **Save** finalizes and publishes the supplied bytes without validating or rebuilding the archive. Publishing a custom ZIP retains the saved builder configuration and banner, so publishing from the builder again does not require another banner upload. Only the active archive is served to Munki clients.

### Browser uploads to S3 need bucket CORS

The `s3` upload goes from the browser directly to the bucket, which makes it a cross-origin request. The bucket has to return CORS headers for the admin origin, or the browser blocks the `PUT`. Multipart uploads also require the browser to read each part's `ETag`. Add a CORS rule that allows the Woodstar origin with `PUT`, `GET`, and `HEAD` and exposes `ETag`:

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

## Backend integration test

The dependency-free Go suite runs the shared storage contract against the file backend. `mise run test-integration-storage` runs the same contract against S3 and starts an ephemeral [Garage](https://garagehq.deuxfleurs.fr/) container with testcontainers. Garage is a small standards-compliant S3 server for the provider test, not a persistent development dependency.

A real deployment supplies its own S3-compatible bucket, credentials, and endpoints. The full provider-agnostic settings are in [Environment](./environment#storage).
