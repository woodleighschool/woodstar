---
sidebar_position: 1
title: Environment
description: The WOODSTAR_ environment variables, their defaults, and what they switch on.
---

# Environment

Woodstar reads its settings from environment variables with a `WOODSTAR_` prefix. CLI flags populate the same config first; environment parsing fills unset fields and applies defaults. Woodstar then normalizes and validates the resolved config independently of either input source.

Three settings are required and the server won't start without them: `WOODSTAR_URL`, `WOODSTAR_DATABASE_URL`, and `WOODSTAR_SESSION_SECRET`.

Several features stay off until you configure them. OIDC and Entra directory sync each switch on only once their settings are complete, so an unset block means that feature is simply not running. Storage is the exception: it always runs, defaulting to local files until you point it at an S3 bucket.

## Server

| Variable                         | Default   | Notes                                                                                                 |
| -------------------------------- | --------- | ----------------------------------------------------------------------------------------------------- |
| `WOODSTAR_HOST`                  | `0.0.0.0` | Listen host.                                                                                          |
| `WOODSTAR_PORT`                  | `8080`    | Listen port.                                                                                          |
| `WOODSTAR_URL`                   | required  | Canonical HTTPS origin used by the app, agents, enrollment profiles, and file-storage redirects.      |
| `WOODSTAR_TLS_CERT_FILE`         | empty     | PEM certificate chain for direct TLS termination. Must be set with `WOODSTAR_TLS_KEY_FILE`.           |
| `WOODSTAR_TLS_KEY_FILE`          | empty     | PEM private key for direct TLS termination. Must be set with `WOODSTAR_TLS_CERT_FILE`.                |
| `WOODSTAR_SESSION_SECRET`        | required  | At least 32 characters.                                                                               |
| `WOODSTAR_SESSION_COOKIE_SECURE` | `true`    | Whether browser session cookies carry the `Secure` attribute. Set `false` only for HTTP development.  |
| `WOODSTAR_DATABASE_URL`          | required  | Postgres connection URL, e.g. `postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable`. |
| `WOODSTAR_LOG_LEVEL`             | `info`    | `debug`, `info`, `warn`, or `error`.                                                                  |

`WOODSTAR_URL` has its trailing slash trimmed and rejects HTTP, sub-paths, credentials, query strings, and fragments. If you need to serve Woodstar under a path, put a reverse proxy in front; the app expects to own the root of its host.

Certificate files are optional because the normal container deployment terminates TLS at a reverse proxy. Set neither file for that private HTTP hop. Set both files when the Woodstar process listens directly on HTTPS. A partial pair is a startup error, and Woodstar does not fall back to HTTP when a configured certificate cannot be loaded.

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

OIDC turns on only when the issuer URL, client ID, and client secret are all set. A partial block is a startup error.

| Variable                      | Default                | Notes                                |
| ----------------------------- | ---------------------- | ------------------------------------ |
| `WOODSTAR_OIDC_ISSUER_URL`    | empty                  | Provider issuer URL.                 |
| `WOODSTAR_OIDC_CLIENT_ID`     | empty                  | Client ID.                           |
| `WOODSTAR_OIDC_CLIENT_SECRET` | empty                  | Client secret.                       |
| `WOODSTAR_OIDC_REDIRECT_URL`  | server callback        | Exact browser callback URL.          |
| `WOODSTAR_OIDC_SCOPES`        | `openid,email,profile` | Scopes to request.                   |
| `WOODSTAR_OIDC_EMAIL_CLAIM`   | `email`                | Claim used as the Woodstar identity. |

The redirect defaults to `<WOODSTAR_URL>/api/auth/sso/callback`. Set it explicitly when the browser reaches the callback through another origin, such as `http://localhost:5173/api/auth/sso/callback` through Vite. HTTPS is required except for loopback HTTP development, and the path must remain `/api/auth/sso/callback`. Configured OIDC discovery must succeed during startup; Woodstar does not silently disable a requested login path.

## Entra directory sync

Sync starts only when the tenant ID, client ID, and client secret are all set. A partial block is a startup error.

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
| `WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT` | empty          | HTTPS endpoint used in presigned URLs. Required when the internal endpoint uses HTTP.  |
| `WOODSTAR_STORAGE_S3_ACCESS_KEY`      | empty          | Access key.                                                                            |
| `WOODSTAR_STORAGE_S3_SECRET_KEY`      | empty          | Secret key.                                                                            |
| `WOODSTAR_STORAGE_S3_PATH_STYLE`      | `false`        | Use path-style addressing.                                                             |
| `WOODSTAR_STORAGE_S3_PRESIGN_TTL`     | `15m`          | Lifetime of presigned URLs. Must be positive.                                          |

With `s3`, the bucket, region, access key, and secret key have to be present together. See [Munki Storage](./storage) for how the backends fit the artifact flow and the bucket CORS rule browser uploads need.

## Distribution point worker

These configure the `woodstar mdp` worker, not the server. The worker is a separate process with its own `WOODSTAR_MDP_` prefix and none of the server's database, session, or storage settings; it talks to Woodstar over the network and authenticates with one distribution point's key. What it does is covered in [Munki Distribution Points](../agent-protocols/munki-distribution).

| Variable                            | Default  | Notes                                                                |
| ----------------------------------- | -------- | -------------------------------------------------------------------- |
| `WOODSTAR_MDP_SERVER_URL`           | required | HTTPS Woodstar origin the worker connects to.                        |
| `WOODSTAR_MDP_SERVER_CA_FILE`       | empty    | PEM CA file for a private Woodstar certificate chain.                |
| `WOODSTAR_MDP_KEY`                  | required | The distribution point's key, from its create or rotate response.    |
| `WOODSTAR_MDP_DATA_DIR`             | required | Directory the mirrored installers and state snapshot live in.        |
| `WOODSTAR_MDP_LISTEN_ADDR`          | `:8080`  | Address the worker serves Macs on.                                   |
| `WOODSTAR_MDP_TLS_CERT_FILE`        | empty    | PEM certificate chain for direct TLS. Must be set with the key file. |
| `WOODSTAR_MDP_TLS_KEY_FILE`         | empty    | PEM private key for direct TLS. Must be set with the certificate.    |
| `WOODSTAR_MDP_LOG_LEVEL`            | `info`   | `debug`, `info`, `warn`, or `error`.                                 |
| `WOODSTAR_MDP_DOWNLOAD_CONCURRENCY` | `4`      | Concurrent installer downloads. Must be at least one.                |

`WOODSTAR_MDP_SERVER_CA_FILE` controls trust for both the HTTPS download client and the WebSocket connection. Set both worker TLS files when the worker terminates HTTPS itself. Leave both empty behind a reverse proxy. The distribution point's client base URL must always be the public HTTPS origin that Macs use.

## A local example

```bash
WOODSTAR_HOST=0.0.0.0
WOODSTAR_PORT=8443
WOODSTAR_URL=https://localhost:8443
WOODSTAR_TLS_CERT_FILE=./tmp/tls/woodstar.pem
WOODSTAR_TLS_KEY_FILE=./tmp/tls/woodstar-key.pem
WOODSTAR_DATABASE_URL=postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
WOODSTAR_SESSION_SECRET=replace-with-at-least-32-characters
WOODSTAR_LOG_LEVEL=debug
```
