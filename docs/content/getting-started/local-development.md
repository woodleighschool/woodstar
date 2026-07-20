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

`mise run dev` runs `dev-tls` first. It creates a repo-local CA and regenerates the server certificate for `woodstar` under `tmp/tls/`. Add that development hostname once on each machine that needs to reach the server:

```bash
# Development host
127.0.0.1 woodstar

# UTM client VM
192.168.64.1 woodstar
```

Use the first entry in the development host's `/etc/hosts` and the second in the VM's `/etc/hosts`. `WOODSTAR_URL` stays `https://woodstar:8443` on the host; the two machines resolve that same service identity to the address they can reach.

The CA certificate is `tmp/tls/ca/rootCA.pem`. Its private key stays under the ignored `tmp/` directory. Regenerating the leaf certificate keeps the same repo-local CA, so clients only need that CA installed once.

## Start Postgres

The checked-in compose file provides only local PostgreSQL:

```bash
docker compose up -d postgres
```

Woodstar uses file storage by default. The storage integration test starts an ephemeral Garage container when it exercises the S3 backend; it is not part of the development stack.

Create a local administrator after Postgres is available. This also runs pending database migrations:

```bash
WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
  mise exec -- go run ./cmd/woodstar user create \
  --email admin@example.com \
  --name Administrator \
  --role admin \
  --password woodstar-development
```

## Run the server

The server needs its canonical HTTPS URL, database URL, and a 32-byte file-storage capability key. A local run usually looks like this:

```bash
mise run dev-tls
mise //web:build

WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_PORT=8443 \
WOODSTAR_URL='https://woodstar:8443' \
WOODSTAR_TLS_CERT_FILE='./tmp/tls/woodstar.pem' \
WOODSTAR_TLS_KEY_FILE='./tmp/tls/woodstar-key.pem' \
WOODSTAR_STORAGE_CAPABILITY_KEY="$(openssl rand -hex 32)" \
  mise exec -- go run ./cmd/woodstar serve
```

This development command listens on `0.0.0.0:8443` and serves HTTPS. Leave both TLS file settings empty only when a reverse proxy terminates HTTPS in front of Woodstar.

## The dev loop

```bash
mise run dev
```

That ensures the development certificate and embedded frontend exist, then runs Vite on plain HTTP and the Go backend on HTTPS. `dev-backend` loads `.env` if it's there and runs Go through Air. Vite serves only the SPA and proxies `/api` to `https://woodstar:8443`; Node receives the repo-local CA through `NODE_EXTRA_CA_CERTS`. You can run either side on its own after generating the certificate and frontend build:

```bash
mise run dev-backend
mise //web:dev
```

Open `https://woodstar:8443` for the normal app and authentication flow. Use `http://localhost:5173` when working through Vite. A browser session through Vite needs `WOODSTAR_SESSION_COOKIE_SECURE=false`; OIDC through Vite also needs `WOODSTAR_OIDC_REDIRECT_URL=http://localhost:5173/api/auth/sso/callback`. Neither override is part of the example environment.

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
| Munki         | Trust the CA with a certificate profile                          |
| Santa         | Trust the CA with a certificate profile                          |
| MDP worker    | `WOODSTAR_MDP_SERVER_CA_FILE=/path/to/woodstar-ca.pem`           |
| AutoPkg       | `WOODSTAR_CA_FILE=/path/to/woodstar-ca.pem`                      |

Munki and Santa use macOS trust evaluation. Deliver the CA as a trusted certificate payload in the same development profile as their settings, or install it interactively on a throwaway client. `SoftwareRepoCACertificate` and Santa's `ServerAuthRootsFile` can identify or pin the CA, but they do not replace the system trust required by current clients. Modern macOS also requires System Settings for manual profile installation, so SSH bootstrap should not try to bypass that approval.

System trust remains opt-in. `mise run dev-tls-trust` trusts the CA only on the development machine where the command runs; it does not change a test VM.

## Local administrator

The administrator created above is an ordinary directory user with an Account page and API-key support. If local development reaches a state with no administrator, rerun `woodstar user set-role --email admin@example.com --role admin` against the same database.
