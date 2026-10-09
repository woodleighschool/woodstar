---
sidebar_position: 2
title: Munki Storage
description: Store and serve Munki installers, icons, and client resources.
---

# Munki Storage

The server stores Munki installers, icons, banners, and client-resource archives in local files or an S3-compatible bucket. Their metadata remains in PostgreSQL.

## File storage

File storage is the default. Set:

```bash
WOODSTAR_STORAGE_KIND=file
WOODSTAR_STORAGE_FILE_ROOT=/var/lib/woodstar/storage
WOODSTAR_STORAGE_CAPABILITY_KEY=<output of openssl rand -hex 32>
```

The server writes files below the configured root and serves them through short-lived `/storage/*` URLs.

## S3 storage

Set `WOODSTAR_STORAGE_KIND=s3` with a bucket, region, access key, and secret key. An endpoint can be supplied for services such as Garage, MinIO, or Cloudflare R2. The server uses that endpoint for its own S3 requests and presigned transfer URLs.

Installer uploads up to 100 MiB use a presigned `PUT`. Larger installers use S3 multipart upload: the server owns the upload lifecycle and presigns each part `PUT`, while the client returns the uploaded part ETags for server-side completion. Downloads use presigned URLs.

### Bucket CORS

Uploads from the web app go directly to the bucket. Allow the server origin to use `GET`, `PUT`, and `HEAD`, and expose `ETag` for multipart uploads:

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

Stemma does not need this browser CORS rule.

## Package uploads

Upload the installer before creating a `pkg` or `copy_from_dmg` package. The server verifies immutable uploaded bytes in a background job before recording their size and SHA-256. `nopkg` packages do not have an installer.

`PUT /api/munki/package-installers/{id}` starts verification once and returns `202` with `Retry-After` while work is queued, running, or retrying. Poll the same endpoint until it returns `200` with verified metadata. A terminal failure returns `422`; another upload is required. Stemma and the web app wait for verification before creating a package.

Verification uses a separate queue with one worker per server, up to three attempts, and a one-hour limit per attempt. Active jobs retain their uploads against deletion and orphan cleanup. Client disconnection does not cancel verification. Unattached uploads remain eligible for orphan cleanup after verification ends. Storage logs report copy and inspection durations separately.

Icons and Client Resources use the same configured backend. Client Resources accepts a banner through its builder or a complete ZIP archive.

## Downloads

Munki continues to request stable repository paths:

- `/munki/pkgs/*`
- `/munki/icons/*`
- `/munki/client_resources/*`

With file storage, the server streams the bytes. With S3, it redirects to a presigned URL. A matching package request can instead be redirected to a [distribution-point cache](../agent-protocols/munki-distribution).

See [Environment](./environment#storage) for every storage setting.
