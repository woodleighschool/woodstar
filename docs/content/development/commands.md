---
sidebar_position: 1
title: Commands
description: Build, test, lint, format, and generation commands for Woodstar.
---

# Commands

Use mise tasks as the repo contract. Direct `go test` or `pnpm` commands are fine while narrowing one failure, but handoff should use the broadest relevant task.

## Setup

```bash
mise trust
mise install
mise run deps
```

`mise run deps` downloads Go modules and installs frontend dependencies in `web/`.

## Build

```bash
mise run build
mise run backend
mise //web:build
```

`build` runs the frontend build first, then builds the Go binary at `./woodstar`.

## Development

```bash
mise run dev
mise run dev-backend
mise run dev-tls
mise run dev-tls-trust
mise //web:dev
```

`dev` builds the embedded frontend and depends on `dev-tls`, which creates repo-local certificate files. `dev-tls-trust` trusts that CA locally. `dev-backend` loads `.env` if present and starts Air; the web task runs Vite from `web/`.

## Tests

```bash
mise run test
mise run full-test
mise run test-integration
mise run test-openapi
```

`mise run test` unsets `WOODSTAR_TEST_DATABASE_URL`, so DB-backed `dbtest` tests skip. `mise run full-test` supplies a local Postgres URL if the variable is unset and expects the database to be reachable.

## Lint And Format

```bash
mise run lint
mise run go-lint
mise //web:lint
mise run format
mise run go-format
mise //web:format
```

The split matters. Backend and frontend checks have different tools and should stay runnable independently.

## Generated Artifacts

```bash
mise run openapi
mise run openapi-types
mise run generate
```

`generate` runs OpenAPI and frontend client generation.

## Local Gate

```bash
mise run precommit
```

The precommit task runs format, lint, full DB-backed tests, and OpenAPI freshness checks.
