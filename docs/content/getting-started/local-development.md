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

The server needs `WOODSTAR_PUBLIC_URL` and a session secret of at least 32 characters. A local run usually looks like this:

```bash
WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_PUBLIC_URL='http://localhost:8080' \
WOODSTAR_SESSION_SECRET='replace-with-at-least-32-characters' \
  mise exec -- go run ./cmd/woodstar serve
```

It listens on `0.0.0.0:8080` by default.

## The dev loop

```bash
mise run dev
```

That starts both sides at once. `dev-backend` loads `.env` if it's there and runs the Go server through Air; `dev-frontend` runs Vite from `web/`. You can run either on its own:

```bash
mise run dev-backend
mise run dev-frontend
```

## First admin account

On a fresh database there are no accounts yet. Start the backend, open the app, and the setup flow walks you through creating the first local admin. From there, sign-in works as described in [Authentication](../configuration/authentication).
