---
sidebar_position: 1
title: Environment
description: Woodstar server and distribution-point settings.
---

# Environment

Woodstar reads settings from `WOODSTAR_` environment variables. `WOODSTAR_URL` and `WOODSTAR_DATABASE_URL` are required. File storage also requires `WOODSTAR_STORAGE_CAPABILITY_KEY`.

OIDC and Entra sync remain disabled when their settings are empty. Supplying only part of either configuration is an error.

## Server

| Variable                         | Default   | Description                                                           |
| -------------------------------- | --------- | --------------------------------------------------------------------- |
| `WOODSTAR_HOST`                  | `0.0.0.0` | Listen address                                                        |
| `WOODSTAR_PORT`                  | `8080`    | Listen port                                                           |
| `WOODSTAR_URL`                   | required  | Public HTTPS origin used by the app and clients                       |
| `WOODSTAR_TLS_CERT_FILE`         | empty     | Certificate chain for direct TLS; set with `WOODSTAR_TLS_KEY_FILE`    |
| `WOODSTAR_TLS_KEY_FILE`          | empty     | Private key for direct TLS; set with `WOODSTAR_TLS_CERT_FILE`         |
| `WOODSTAR_SESSION_COOKIE_SECURE` | `true`    | Add the `Secure` attribute to session cookies                         |
| `WOODSTAR_DATABASE_URL`          | required  | PostgreSQL connection URL                                             |
| `WOODSTAR_LOG_LEVEL`             | `info`    | `debug`, `info`, `warn`, or `error`                                   |
| `WOODSTAR_CORS_ALLOWED_ORIGINS`  | empty     | Comma-separated web origins allowed to make credentialed API requests |

`WOODSTAR_URL` must be an HTTPS origin without credentials, a query, a fragment, or a sub-path. Set both TLS files when Woodstar terminates HTTPS. Leave both empty when a reverse proxy handles TLS.

Set `WOODSTAR_SESSION_COOKIE_SECURE=false` only for HTTP development. CORS origins must be origins such as `http://localhost:5173`, not URLs with paths.

## Client IP

Woodstar derives a client IP from each package request to select a [Munki distribution point](../agent-protocols/munki-distribution#how-client-matching-works). The default reads the connection address. Choose a proxy-aware source when TLS or HTTP terminates elsewhere.

| Variable                                      | Default       | Description                                                            |
| --------------------------------------------- | ------------- | ---------------------------------------------------------------------- |
| `WOODSTAR_HTTP_CLIENT_IP_SOURCE`              | `remote_addr` | `remote_addr`, `xff_trusted_cidrs`, `xff_trusted_proxies`, or `header` |
| `WOODSTAR_HTTP_CLIENT_IP_TRUSTED_CIDRS`       | empty         | Trusted proxy CIDRs for `xff_trusted_cidrs`                            |
| `WOODSTAR_HTTP_CLIENT_IP_TRUSTED_PROXY_COUNT` | empty         | Number of proxies for `xff_trusted_proxies`                            |
| `WOODSTAR_HTTP_CLIENT_IP_HEADER`              | empty         | Trusted header name for `header`                                       |

Each non-default mode requires its matching setting. Use `xff_trusted_cidrs` for known proxy networks, `xff_trusted_proxies` for a fixed proxy chain, or `header` when the proxy supplies a dedicated client-IP header.

Only trust forwarded addresses when the Woodstar origin is restricted to that proxy path and the proxy replaces or sanitizes the selected header. If clients can reach the origin directly, they can otherwise supply a false source address and affect distribution-point selection.

## Santa event retention

| Variable                              | Default | Description                         |
| ------------------------------------- | ------- | ----------------------------------- |
| `WOODSTAR_SANTA_EVENT_RETENTION_DAYS` | `90`    | Number of days to keep Santa events |
| `WOODSTAR_SANTA_EVENT_SWEEP_INTERVAL` | `1h`    | Cleanup interval                    |

## OIDC

| Variable                      | Default                | Description                                |
| ----------------------------- | ---------------------- | ------------------------------------------ |
| `WOODSTAR_OIDC_ISSUER_URL`    | empty                  | Provider issuer URL                        |
| `WOODSTAR_OIDC_CLIENT_ID`     | empty                  | Client ID                                  |
| `WOODSTAR_OIDC_CLIENT_SECRET` | empty                  | Client secret                              |
| `WOODSTAR_OIDC_REDIRECT_URL`  | server callback        | Browser callback URL                       |
| `WOODSTAR_OIDC_SCOPES`        | `openid,email,profile` | Comma-separated scopes                     |
| `WOODSTAR_OIDC_EMAIL_CLAIM`   | `email`                | Claim matched to the Woodstar user's email |

The default redirect is `<WOODSTAR_URL>/api/auth/sso/callback`. An override must use the same path and HTTPS, except that loopback HTTP is accepted for development.

## Entra directory sync

| Variable                           | Default | Description                    |
| ---------------------------------- | ------- | ------------------------------ |
| `WOODSTAR_ENTRA_TENANT_ID`         | empty   | Tenant ID                      |
| `WOODSTAR_ENTRA_CLIENT_ID`         | empty   | Application client ID          |
| `WOODSTAR_ENTRA_CLIENT_SECRET`     | empty   | Application client secret      |
| `WOODSTAR_ENTRA_TRANSITIVE_GROUPS` | `false` | Expand nested group membership |
| `WOODSTAR_ENTRA_SYNC_INTERVAL`     | `1h`    | Sync interval                  |

See [Directory](../admin/directory) for the data Woodstar imports.

## Storage

| Variable                              | Default             | Description                                          |
| ------------------------------------- | ------------------- | ---------------------------------------------------- |
| `WOODSTAR_STORAGE_KIND`               | `file`              | `file` or `s3`                                       |
| `WOODSTAR_STORAGE_FILE_ROOT`          | `data/storage`      | Storage directory for `file`                         |
| `WOODSTAR_STORAGE_CAPABILITY_KEY`     | required for `file` | 32 random bytes encoded as 64 hexadecimal characters |
| `WOODSTAR_STORAGE_TRANSFER_TTL`       | `15m`               | Lifetime of file capabilities and S3 presigned URLs  |
| `WOODSTAR_STORAGE_S3_BUCKET`          | empty               | S3 bucket                                            |
| `WOODSTAR_STORAGE_S3_REGION`          | empty               | S3 region                                            |
| `WOODSTAR_STORAGE_S3_ENDPOINT`        | empty               | S3 endpoint; leave empty for AWS                     |
| `WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT` | empty               | HTTPS endpoint used in presigned URLs                |
| `WOODSTAR_STORAGE_S3_ACCESS_KEY`      | empty               | S3 access key                                        |
| `WOODSTAR_STORAGE_S3_SECRET_KEY`      | empty               | S3 secret key                                        |
| `WOODSTAR_STORAGE_S3_PATH_STYLE`      | `false`             | Use path-style bucket URLs                           |

Generate a file-storage key with:

```bash
openssl rand -hex 32
```

The S3 bucket, region, access key, and secret key are required together. See [Munki Storage](./storage) for upload and download behaviour.

## Distribution point worker

These settings belong to the separate `woodstar mdp` process.

| Variable                            | Default  | Description                                                 |
| ----------------------------------- | -------- | ----------------------------------------------------------- |
| `WOODSTAR_MDP_SERVER_URL`           | required | Woodstar HTTPS origin                                       |
| `WOODSTAR_MDP_SERVER_CA_FILE`       | empty    | Additional CA certificate trusted by the worker             |
| `WOODSTAR_MDP_KEY`                  | required | Key shown when the distribution point is created or rotated |
| `WOODSTAR_MDP_DATA_DIR`             | required | Directory for cached installers and worker state            |
| `WOODSTAR_MDP_LISTEN_ADDR`          | `:8080`  | Address that serves package downloads                       |
| `WOODSTAR_MDP_TLS_CERT_FILE`        | empty    | Certificate chain for direct TLS; set with the key file     |
| `WOODSTAR_MDP_TLS_KEY_FILE`         | empty    | Private key for direct TLS; set with the certificate file   |
| `WOODSTAR_MDP_LOG_LEVEL`            | `info`   | `debug`, `info`, `warn`, or `error`                         |
| `WOODSTAR_MDP_DOWNLOAD_CONCURRENCY` | `4`      | Number of concurrent installer downloads                    |

The client-facing cache URL is configured on the distribution-point record in Woodstar. The URL must use HTTPS and be reachable from every client CIDR assigned to the point. Woodstar does not test that client-side route.

## Example

The repository's `.env.example` contains a complete local configuration. At minimum, fill in `WOODSTAR_STORAGE_CAPABILITY_KEY` before starting the server.
