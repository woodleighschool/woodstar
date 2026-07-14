---
sidebar_position: 1
title: Local Development
description: Run Woodstar from a checkout with mise, Postgres, and the frontend toolchain.
---

# Local Development

Woodstar uses [mise](https://mise.jdx.dev/) tasks as the repo's contract. Reach for those first: they keep Go, Node, pnpm, the linters, generated code, and the test commands in one place.

## Prerequisites

```bash
brew install mise
mise trust
mise install
mise run deps
```

`mise run deps` pulls the Go modules and installs the frontend dependencies under `web/`.

`mise run dev` runs `dev-tls` first. It creates a repo-local CA and server certificate under `tmp/tls/`. Run it directly to add an address used by an external test client:

```bash
mise run dev-tls -- 192.168.64.1
```

The CA certificate is `tmp/tls/ca/rootCA.pem`. Its private key stays under the ignored `tmp/` directory.

## Start Postgres

The checked-in compose file gives you local Postgres, plus Garage for Munki artifact work:

```bash
docker compose up -d postgres
```

If you're going to touch Munki artifacts, start Garage too:

```bash
docker compose up -d garage
```

## Run the server

The server needs its canonical HTTPS URL, database URL, and a session secret of at least 32 characters. A local run usually looks like this:

```bash
mise run dev-tls
mise //web:build

WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_PORT=8443 \
WOODSTAR_URL='https://localhost:8443' \
WOODSTAR_TLS_CERT_FILE='./tmp/tls/woodstar.pem' \
WOODSTAR_TLS_KEY_FILE='./tmp/tls/woodstar-key.pem' \
WOODSTAR_SESSION_SECRET='replace-with-at-least-32-characters' \
  mise exec -- go run ./cmd/woodstar serve
```

This development command listens on `0.0.0.0:8443` and serves HTTPS. Leave both TLS file settings empty only when a reverse proxy terminates HTTPS in front of Woodstar.

## The dev loop

```bash
mise run dev
```

That ensures the development certificate and embedded frontend exist, then runs Vite on plain HTTP and the Go backend on HTTPS. `dev-backend` loads `.env` if it's there and runs Go through Air. Vite serves only the SPA and proxies `/api` to `https://localhost:8443`; Node receives the repo-local CA through `NODE_EXTRA_CA_CERTS`. You can run either side on its own after generating the certificate and frontend build:

```bash
mise run dev-backend
mise //web:dev
```

Open `https://localhost:8443` for the normal app and authentication flow. Use `http://localhost:5173` when working through Vite. A browser session through Vite needs `WOODSTAR_SESSION_COOKIE_SECURE=false`; OIDC through Vite also needs `WOODSTAR_OIDC_REDIRECT_URL=http://localhost:5173/api/auth/sso/callback`. Neither override is part of the example environment.

## Development certificate trust

Vite already receives the CA file. To trust it locally:

```bash
mise run dev-tls-trust
```

For local agents, bundle `tmp/tls/ca/rootCA.pem` into the Orbit package or copy it to the client that needs it:

| Client        | Development setting                                              |
| ------------- | ---------------------------------------------------------------- |
| Orbit package | `fleetctl package ... --fleet-certificate=tmp/tls/ca/rootCA.pem` |
| osquery       | `--tls_server_certs=/path/to/woodstar-ca.pem`                    |
| Munki         | `SoftwareRepoCACertificate=/path/to/woodstar-ca.pem`             |
| Santa         | `ServerAuthRootsFile=/path/to/woodstar-ca.pem`                   |

For short-lived local testing, a client's explicit insecure mode, such as Fleet's `--insecure`, is another option.

## First admin account

On a fresh database there are no accounts yet. Start the backend, open the app, and the setup flow walks you through creating the first local admin. From there, sign-in works as described in [Authentication](../configuration/authentication).
