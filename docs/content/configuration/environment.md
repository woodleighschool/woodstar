---
sidebar_position: 1
title: Environment
description: Runtime configuration loaded from WOODSTAR_ environment variables.
---

# Environment

Woodstar loads runtime settings with the `WOODSTAR_` prefix. CLI flags populate the same config struct, then environment parsing applies defaults and validation.

`WOODSTAR_PUBLIC_URL` and `WOODSTAR_SESSION_SECRET` are required.

## Server

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_HOST` | `0.0.0.0` | HTTP listen host. |
| `WOODSTAR_PORT` | `8080` | HTTP listen port. |
| `WOODSTAR_PUBLIC_URL` | required | Must be `http` or `https`, include a host, and omit path, query, and fragment. |
| `WOODSTAR_SESSION_SECRET` | required | Must be at least 32 characters. |
| `WOODSTAR_DATABASE_URL` | empty | Postgres connection URL. Local examples use `postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable`. |
| `WOODSTAR_LOG_LEVEL` | `info` | Parsed by Woodstar's logging package. |
| `WOODSTAR_SHUTDOWN_TIMEOUT_SECONDS` | `15` | Graceful shutdown timeout. |

`WOODSTAR_PUBLIC_URL` is normalized by trimming a trailing slash. Sub-path hosting is rejected by config validation; use a reverse proxy if a deployment needs path rewriting.

## Santa Retention

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` | `90` | Retention window for Santa event cleanup. |
| `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL` | `1h` | Cleanup loop interval. |

## OIDC

OIDC is enabled only when all three required OIDC settings are present:

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_OIDC_ISSUER_URL` | empty | Provider issuer URL. |
| `WOODSTAR_OIDC_CLIENT_ID` | empty | Client ID. |
| `WOODSTAR_OIDC_CLIENT_SECRET` | empty | Client secret. |
| `WOODSTAR_OIDC_SCOPES` | `openid,email,profile` | Scope list parsed from the environment. |
| `WOODSTAR_OIDC_EMAIL_CLAIM` | `email` | Claim used as the Woodstar email identity. |

The redirect URL is built as:

```text
<WOODSTAR_PUBLIC_URL>/api/auth/sso/callback
```

If discovery fails at startup, Woodstar logs a warning and disables SSO for that run.

## Entra Directory Sync

Entra sync starts only when tenant ID, client ID, and client secret are configured.

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_ENTRA_TENANT_ID` | empty | Entra tenant ID. |
| `WOODSTAR_ENTRA_CLIENT_ID` | empty | Entra app client ID. |
| `WOODSTAR_ENTRA_CLIENT_SECRET` | empty | Entra app client secret. |
| `WOODSTAR_ENTRA_TRANSITIVE_GROUPS` | `false` | Whether group expansion should be transitive. |
| `WOODSTAR_ENTRA_SYNC_INTERVAL` | `1h` | Directory sync schedule. |

Directory data can feed derived labels and user-affinity enrichment. The docs do not assume a particular tenant shape.

## Munki S3 Storage

Munki artifact upload and redirect flows use S3-compatible storage when configured.

| Variable | Default | Notes |
| --- | --- | --- |
| `WOODSTAR_MUNKI_S3_BUCKET` | empty | Bucket name. |
| `WOODSTAR_MUNKI_S3_REGION` | empty | Region value passed to the S3 client. |
| `WOODSTAR_MUNKI_S3_ENDPOINT` | empty | Internal S3 endpoint. |
| `WOODSTAR_MUNKI_S3_PUBLIC_ENDPOINT` | empty | Public endpoint used for generated artifact URLs. |
| `WOODSTAR_MUNKI_S3_ACCESS_KEY` | empty | Access key. |
| `WOODSTAR_MUNKI_S3_SECRET_KEY` | empty | Secret key. |
| `WOODSTAR_MUNKI_S3_PATH_STYLE` | `false` | Path-style addressing toggle. |
| `WOODSTAR_MUNKI_S3_PRESIGN_TTL` | `15m` | Must be positive. |

If any Munki S3 field is present, the required storage fields must be complete. The current code treats bucket, region, access key, and secret key as required for S3 to be enabled.

## Example

```bash
WOODSTAR_HOST=0.0.0.0
WOODSTAR_PORT=8080
WOODSTAR_PUBLIC_URL=http://localhost:8080
WOODSTAR_DATABASE_URL=postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
WOODSTAR_SESSION_SECRET=replace-with-at-least-32-characters
WOODSTAR_LOG_LEVEL=debug
```
