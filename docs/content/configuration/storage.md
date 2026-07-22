---
sidebar_position: 2
title: Munki Storage
description: Store and serve Munki installers, icons, and client resources.
---

# Munki Storage

Woodstar stores Munki installers, icons, banners, and client-resource archives in local files or an S3-compatible bucket. Their metadata remains in PostgreSQL.

## File storage

File storage is the default. Set:

```bash
WOODSTAR_STORAGE_KIND=file
WOODSTAR_STORAGE_FILE_ROOT=/var/lib/woodstar/storage
WOODSTAR_STORAGE_CAPABILITY_KEY=<output of openssl rand -hex 32>
```

Woodstar writes the files below the configured root and serves them through short-lived `/storage/*` URLs.

## S3 storage

Set `WOODSTAR_STORAGE_KIND=s3` with a bucket, region, access key, and secret key. An endpoint can be supplied for services such as Garage, MinIO, or Cloudflare R2. Use `WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT` when the public download endpoint differs from the one Woodstar connects to.

Woodstar uses multipart upload with presigned part URLs for S3 uploads. Downloads use presigned URLs.

### Bucket CORS

Uploads from the web app go directly to the bucket. Allow the Woodstar origin to use `GET`, `PUT`, and `HEAD`, and expose `ETag` for multipart uploads:

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

AutoPkg does not need this browser CORS rule.

## Package uploads

Upload the installer before creating a `pkg` or `copy_from_dmg` package. Woodstar records its size and SHA-256 when the upload is finalized. `nopkg` packages do not have an installer.

Icons and Client Resources use the same configured backend. Client Resources accepts a banner through its builder or a complete ZIP archive.

## Downloads

Munki continues to request stable repository paths:

- `/munki/pkgs/*`
- `/munki/icons/*`
- `/munki/client_resources/*`

With file storage, Woodstar streams the bytes. With S3, Woodstar redirects to a presigned URL. A matching package request can instead be redirected to a [distribution-point cache](../agent-protocols/munki-distribution).

See [Environment](./environment#storage) for every storage setting.
