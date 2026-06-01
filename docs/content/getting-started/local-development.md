---
sidebar_position: 1
title: Local Development
description: Run Woodstar from a checkout with mise, Postgres, and the frontend toolchain.
---

# Local Development

Woodstar uses mise tasks as the repository contract. Use those tasks first. They keep Go, Node, pnpm, linters, generated code, and test commands in one place.

## Prerequisites

```bash
brew install mise
mise trust
mise install
mise run deps
```

`mise run deps` downloads Go modules and installs the frontend dependencies in `web/`.

## Start Postgres

The checked-in compose file provides local Postgres and Garage for Munki artifact work:

```bash
docker compose up -d postgres
```

For Munki artifact flows, start Garage as well:

```bash
docker compose up -d garage
```

## Run The Server

The server requires `WOODSTAR_PUBLIC_URL` and a session secret of at least 32 characters. A local run usually looks like this:

```bash
WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_PUBLIC_URL='http://localhost:8080' \
WOODSTAR_SESSION_SECRET='replace-with-at-least-32-characters' \
  mise exec -- go run ./cmd/woodstar serve
```

The default listen address is `0.0.0.0:8080`.

## Development Loop

```bash
mise run dev-backend
mise run dev-frontend
```

`dev-backend` loads `.env` if it exists, then starts the Go server through Air. `dev-frontend` runs Vite from `web/`.

The aggregate command starts both:

```bash
mise run dev
```

## First Admin Account

The setup route is mounted under `/api/setup`, and the frontend has a `setup` page. Start the backend, open the app, and create the first local admin account there.

OIDC does not replace initial local setup in the current code. OIDC endpoints are enabled only when issuer URL, client ID, and client secret are all configured.
