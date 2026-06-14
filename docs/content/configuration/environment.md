---
sidebar_position: 1
title: Environment
description: The WOODSTAR_ environment variables, their defaults, and what they switch on.
---

# Environment

Woodstar reads its settings from environment variables with a `WOODSTAR_` prefix. CLI flags fill the same config, then environment parsing applies defaults and validation on top.

Two settings are required and the server won't start without them: `WOODSTAR_PUBLIC_URL` and `WOODSTAR_SESSION_SECRET`.

Several features stay off until you configure them. OIDC and Entra directory sync each switch on only once their settings are complete, so an unset block means that feature is simply not running. Storage is the exception: it always runs, defaulting to local files until you point it at an S3 bucket.

## Server

| Variable                  | Default   | Notes                                                                                                 |
| ------------------------- | --------- | ----------------------------------------------------------------------------------------------------- |
| `WOODSTAR_HOST`           | `0.0.0.0` | Listen host.                                                                                          |
| `WOODSTAR_PORT`           | `8080`    | Listen port.                                                                                          |
| `WOODSTAR_PUBLIC_URL`     | required  | Must be `http` or `https`, include a host, and carry no path, query, or fragment.                     |
| `WOODSTAR_SESSION_SECRET` | required  | At least 32 characters.                                                                               |
| `WOODSTAR_DATABASE_URL`   | empty     | Postgres connection URL, e.g. `postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable`. |
| `WOODSTAR_LOG_LEVEL`      | `info`    | Log level.                                                                                            |

`WOODSTAR_PUBLIC_URL` has its trailing slash trimmed and rejects a sub-path. If you need to serve Woodstar under a path, put a reverse proxy in front; the app expects to own the root of its host.

## Santa event retention

| Variable                              | Default | Notes                            |
| ------------------------------------- | ------- | -------------------------------- |
| `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` | `90`    | How long Santa events are kept.  |
| `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL` | `1h`    | How often the cleanup loop runs. |

## OIDC

OIDC turns on only when the issuer URL, client ID, and client secret are all set.

| Variable                      | Default                | Notes                                |
| ----------------------------- | ---------------------- | ------------------------------------ |
| `WOODSTAR_OIDC_ISSUER_URL`    | empty                  | Provider issuer URL.                 |
| `WOODSTAR_OIDC_CLIENT_ID`     | empty                  | Client ID.                           |
| `WOODSTAR_OIDC_CLIENT_SECRET` | empty                  | Client secret.                       |
| `WOODSTAR_OIDC_SCOPES`        | `openid,email,profile` | Scopes to request.                   |
| `WOODSTAR_OIDC_EMAIL_CLAIM`   | `email`                | Claim used as the Woodstar identity. |

The redirect URL is `<WOODSTAR_PUBLIC_URL>/api/auth/sso/callback`. If discovery fails when the server starts, it logs a warning and runs with SSO off for that boot; local sign-in keeps working.

## Entra directory sync

Sync starts only when the tenant ID, client ID, and client secret are all set.

| Variable                           | Default | Notes                           |
| ---------------------------------- | ------- | ------------------------------- |
| `WOODSTAR_ENTRA_TENANT_ID`         | empty   | Tenant ID.                      |
| `WOODSTAR_ENTRA_CLIENT_ID`         | empty   | App client ID.                  |
| `WOODSTAR_ENTRA_CLIENT_SECRET`     | empty   | App client secret.              |
| `WOODSTAR_ENTRA_TRANSITIVE_GROUPS` | `false` | Expand nested group membership. |
| `WOODSTAR_ENTRA_SYNC_INTERVAL`     | `1h`    | How often sync runs.            |

What the sync feeds is covered in [Directory](../admin/directory).

## Storage

Munki package and icon bytes go to the backend chosen by `WOODSTAR_STORAGE_KIND`. It defaults to `file`, so storage works with no configuration. Set it to `s3` for an S3-compatible bucket, which unlocks presigned uploads and redirects.

| Variable                              | Default        | Notes                                                                                  |
| ------------------------------------- | -------------- | -------------------------------------------------------------------------------------- |
| `WOODSTAR_STORAGE_KIND`               | `file`         | `file` or `s3`.                                                                        |
| `WOODSTAR_STORAGE_FILE_ROOT`          | `data/storage` | Root directory for the `file` backend.                                                 |
| `WOODSTAR_STORAGE_S3_BUCKET`          | empty          | Bucket name.                                                                           |
| `WOODSTAR_STORAGE_S3_REGION`          | empty          | Region passed to the S3 client.                                                        |
| `WOODSTAR_STORAGE_S3_ENDPOINT`        | empty          | Internal S3 endpoint. Leave empty for AWS; set it for Garage, R2, MinIO, and the like. |
| `WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT` | empty          | Public endpoint used in presigned URLs.                                                |
| `WOODSTAR_STORAGE_S3_ACCESS_KEY`      | empty          | Access key.                                                                            |
| `WOODSTAR_STORAGE_S3_SECRET_KEY`      | empty          | Secret key.                                                                            |
| `WOODSTAR_STORAGE_S3_PATH_STYLE`      | `false`        | Use path-style addressing.                                                             |
| `WOODSTAR_STORAGE_S3_PRESIGN_TTL`     | `15m`          | Lifetime of presigned URLs. Must be positive.                                          |

With `s3`, the bucket, region, access key, and secret key have to be present together. See [Munki Storage](./storage) for how the backends fit the artifact flow, including the local Garage defaults and the bucket CORS rule browser uploads need.

## A local example

```bash
WOODSTAR_HOST=0.0.0.0
WOODSTAR_PORT=8080
WOODSTAR_PUBLIC_URL=http://localhost:8080
WOODSTAR_DATABASE_URL=postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
WOODSTAR_SESSION_SECRET=replace-with-at-least-32-characters
WOODSTAR_LOG_LEVEL=debug
```
