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

Set `WOODSTAR_STORAGE_KIND=s3` with a bucket, region, access key, and secret key. An endpoint can be supplied for services such as Garage or Cloudflare R2. The server uses that endpoint for its own S3 requests and presigned transfer URLs.

The provider must accept S3 checksum headers on presigned requests and report object checksums on `HeadObject`. Uploads rely on a signed SHA-256 for a single `PUT`, and on full-object CRC64NVME multipart uploads with a signed CRC64NVME for each part. An upload is published only when the provider reports a matching checksum for the stored object.

Uploads use one presigned `PUT`, except installers larger than 100 MiB, which use S3 multipart upload with a presigned `PUT` for each part. Downloads use presigned URLs.

### Bucket CORS

Uploads from the web app go directly to the bucket and carry checksum headers. Allow the server origin to use `GET`, `PUT`, and `HEAD` with any request header:

```json
[
  {
    "AllowedOrigins": ["https://woodstar.example.com"],
    "AllowedMethods": ["GET", "PUT", "HEAD"],
    "AllowedHeaders": ["*"]
  }
]
```

Stemma does not need this browser CORS rule.

## Uploads

Installers, icons, banners, and client-resource archives share one upload flow:

1. Create the upload with its filename and a declaration of its bytes: `size_bytes`, `sha256`, and `crc64nvme`. Both digests are lowercase hexadecimal.
2. Send the bytes to the returned target with the method and headers it lists.
3. Finalize the upload.

Storage accepts only the declared bytes. With file storage, the upload URL carries the declared size and SHA-256, and the server refuses any other body. With S3, the size and SHA-256 are part of a single upload URL's signature, and the bucket refuses any other body.

Finalizing compares storage's record of the stored object with the declaration, detects the content type from the first bytes, and publishes the object without reading it back. It returns the published object, and repeating it returns the same object. When the bytes are missing or differ from the declaration, it returns `400` and deletes the upload.

| Upload                  | Create                                             | Finalize                                                                   |
| ----------------------- | -------------------------------------------------- | -------------------------------------------------------------------------- |
| Installer               | `POST /api/munki/package-installers`               | `PUT /api/munki/package-installers/{id}`                                   |
| Icon                    | `POST /api/munki/icons`                            | `PUT /api/munki/software/{id}/icon`                                        |
| Client Resources banner | `POST /api/munki/client-resources/banner-uploads`  | `POST /api/munki/client-resources`, `PUT /api/munki/client-resources/{id}` |
| Client Resources ZIP    | `POST /api/munki/client-resources/archive-uploads` | `POST /api/munki/client-resources`, `PUT /api/munki/client-resources/{id}` |

Finalizing an icon, banner, or archive also attaches it. Finalize an installer before creating its `pkg` or `copy_from_dmg` package; a package can't reference a pending upload. `nopkg` packages do not have an installer.

Uploads that are never finalized and published objects that are never attached are removed by periodic cleanup.

### Multipart uploads

When creating an installer upload returns the `multipart` strategy, split the file into at most 10,000 parts of equal size, each at least 5 MiB; the last part may be smaller. For each part, request `POST /api/munki/package-installers/{id}/multipart/parts/{part_number}` with that part's `crc64nvme`, then send the part to the returned target. The bucket refuses part bytes that don't match the checksum.

There is no completion request. Finalizing assembles the parts the bucket holds, checks that they total the declared size, and compares the bucket's CRC64NVME of the assembled object with the declaration. The bucket vouches for a multipart installer's size and CRC64NVME; its SHA-256 is recorded as declared.

## Downloads

Munki continues to request stable repository paths:

- `/munki/pkgs/*`
- `/munki/icons/*`
- `/munki/client_resources/*`

With file storage, the server streams the bytes. With S3, it redirects to a presigned URL. A matching package request can instead be redirected to a [distribution-point cache](../agent-protocols/munki-distribution).

See [Environment](./environment#storage) for every storage setting.
