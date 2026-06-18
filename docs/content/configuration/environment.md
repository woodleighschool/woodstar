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

## Client IP

Woodstar needs the real client IP to route a Mac to a [distribution point](../agent-protocols/munki-distribution) by its network. Behind a proxy or CDN the connection's address is the proxy, not the Mac, so you tell Woodstar where to find the true address.

| Variable                                      | Default       | Notes                                                                                         |
| --------------------------------------------- | ------------- | --------------------------------------------------------------------------------------------- |
| `WOODSTAR_HTTP_CLIENT_IP_SOURCE`              | `remote_addr` | `remote_addr`, `xff_trusted_cidrs`, `xff_trusted_proxies`, or `header`.                       |
| `WOODSTAR_HTTP_CLIENT_IP_TRUSTED_CIDRS`       | empty         | Required for `xff_trusted_cidrs`. Proxy prefixes to skip from the right of `X-Forwarded-For`. |
| `WOODSTAR_HTTP_CLIENT_IP_TRUSTED_PROXY_COUNT` | empty         | Required for `xff_trusted_proxies`. How many proxies sit in front, counted from the right.    |
| `WOODSTAR_HTTP_CLIENT_IP_HEADER`              | empty         | Required for `header`. A single trusted header to read, such as `CF-Connecting-IP`.           |

The default trusts the connection's own address, which is right when nothing terminates in front of Woodstar. Pick one of the others when a proxy or CDN does, and set its companion variable. Trusting `X-Forwarded-For` blindly would let a client spoof its own IP, so the trusted modes only believe the parts of the header your own proxies wrote: `xff_trusted_cidrs` skips addresses matching known proxy ranges, `xff_trusted_proxies` skips a fixed count, and `header` ignores `X-Forwarded-For` for a header you know your proxy sets. Each mode refuses to start without its companion.

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

## Distribution point worker

These configure the `woodstar mdp` worker, not the server. The worker is a separate process with its own `WOODSTAR_MDP_` prefix and none of the server's database, session, or storage settings; it talks to Woodstar over the network and authenticates with one distribution point's key. What it does is covered in [Munki Distribution Points](../agent-protocols/munki-distribution).

| Variable                            | Default  | Notes                                                                 |
| ----------------------------------- | -------- | --------------------------------------------------------------------- |
| `WOODSTAR_MDP_SERVER_URL`           | required | The Woodstar base URL the worker connects to.                         |
| `WOODSTAR_MDP_KEY`                  | required | The distribution point's key, from its create or rotate response.     |
| `WOODSTAR_MDP_DATA_DIR`             | required | Directory the mirrored installers and the state snapshot live in.     |
| `WOODSTAR_MDP_LISTEN_ADDR`          | `:8080`  | Address the worker serves Macs on. Front it with a TLS reverse proxy. |
| `WOODSTAR_MDP_LOG_LEVEL`            | `info`   | Log level.                                                            |
| `WOODSTAR_MDP_DOWNLOAD_CONCURRENCY` | `4`      | How many installers the worker downloads at once while catching up.   |

The worker serves plain HTTP and doesn't terminate TLS. Set the distribution point's client base URL to the public HTTPS address of the proxy in front of it.

## A local example

```bash
WOODSTAR_HOST=0.0.0.0
WOODSTAR_PORT=8080
WOODSTAR_PUBLIC_URL=http://localhost:8080
WOODSTAR_DATABASE_URL=postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
WOODSTAR_SESSION_SECRET=replace-with-at-least-32-characters
WOODSTAR_LOG_LEVEL=debug
```
