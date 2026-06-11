---
sidebar_position: 1
title: Environment
description: The WOODSTAR_ environment variables, their defaults, and what they switch on.
---

# Environment

Woodstar reads its settings from environment variables with a `WOODSTAR_` prefix. CLI flags fill the same config, then environment parsing applies defaults and validation on top.

Two settings are required and the server won't start without them: `WOODSTAR_PUBLIC_URL` and `WOODSTAR_SESSION_SECRET`.

Several features stay off until you configure them. OIDC, Entra directory sync, and Munki S3 storage each switch on only once their settings are complete, so an unset block means that feature is simply not running.

## Server

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_HOST` | `0.0.0.0` | Listen host. |
| `WOODSTAR_PORT` | `8080` | Listen port. |
| `WOODSTAR_PUBLIC_URL` | required | Must be `http` or `https`, include a host, and carry no path, query, or fragment. |
| `WOODSTAR_SESSION_SECRET` | required | At least 32 characters. |
| `WOODSTAR_DATABASE_URL` | empty | Postgres connection URL, e.g. `postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable`. |
| `WOODSTAR_LOG_LEVEL` | `info` | Log level. |

`WOODSTAR_PUBLIC_URL` has its trailing slash trimmed and rejects a sub-path. If you need to serve Woodstar under a path, put a reverse proxy in front; the app expects to own the root of its host.

## Santa event retention

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` | `90` | How long Santa events are kept. |
| `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL` | `1h` | How often the cleanup loop runs. |

## OIDC

OIDC turns on only when the issuer URL, client ID, and client secret are all set.

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_OIDC_ISSUER_URL` | empty | Provider issuer URL. |
| `WOODSTAR_OIDC_CLIENT_ID` | empty | Client ID. |
| `WOODSTAR_OIDC_CLIENT_SECRET` | empty | Client secret. |
| `WOODSTAR_OIDC_SCOPES` | `openid,email,profile` | Scopes to request. |
| `WOODSTAR_OIDC_EMAIL_CLAIM` | `email` | Claim used as the Woodstar identity. |

The redirect URL is `<WOODSTAR_PUBLIC_URL>/api/auth/sso/callback`. If discovery fails when the server starts, it logs a warning and runs with SSO off for that boot; local sign-in keeps working.

## Entra directory sync

Sync starts only when the tenant ID, client ID, and client secret are all set.

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_ENTRA_TENANT_ID` | empty | Tenant ID. |
| `WOODSTAR_ENTRA_CLIENT_ID` | empty | App client ID. |
| `WOODSTAR_ENTRA_CLIENT_SECRET` | empty | App client secret. |
| `WOODSTAR_ENTRA_TRANSITIVE_GROUPS` | `false` | Expand nested group membership. |
| `WOODSTAR_ENTRA_SYNC_INTERVAL` | `1h` | How often sync runs. |

What the sync feeds is covered in [Directory](../admin/directory).

## Munki S3 storage

Munki artifact upload and redirect use S3-compatible storage when it's configured. Bucket, region, access key, and secret key are the four that have to be present together; set one part of the block and Woodstar expects the rest.

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_MUNKI_S3_BUCKET` | empty | Bucket name. |
| `WOODSTAR_MUNKI_S3_REGION` | empty | Region passed to the S3 client. |
| `WOODSTAR_MUNKI_S3_ENDPOINT` | empty | Internal S3 endpoint. |
| `WOODSTAR_MUNKI_S3_PUBLIC_ENDPOINT` | empty | Public endpoint used in generated artifact URLs. |
| `WOODSTAR_MUNKI_S3_ACCESS_KEY` | empty | Access key. |
| `WOODSTAR_MUNKI_S3_SECRET_KEY` | empty | Secret key. |
| `WOODSTAR_MUNKI_S3_PATH_STYLE` | `false` | Use path-style addressing. |
| `WOODSTAR_MUNKI_S3_PRESIGN_TTL` | `15m` | Lifetime of presigned URLs. Must be positive. |

See [Munki Storage](./storage) for how this fits the artifact flow, including the local Garage defaults.

## A local example

```bash
WOODSTAR_HOST=0.0.0.0
WOODSTAR_PORT=8080
WOODSTAR_PUBLIC_URL=http://localhost:8080
WOODSTAR_DATABASE_URL=postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
WOODSTAR_SESSION_SECRET=replace-with-at-least-32-characters
WOODSTAR_LOG_LEVEL=debug
```
