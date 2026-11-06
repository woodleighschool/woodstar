---
sidebar_position: 2
title: Commands
description: Build, test, lint, format, and generate the server.
---

# Commands

Run repository tasks through mise.

## Setup

```bash
mise trust
mise install
mise run deps
docker compose up -d postgres
```

## Development and build

```bash
mise run dev
mise run dev-backend
mise run //web:dev
mise run build
mise run backend
mise run geoip
mise run //web:build
```

`mise run build` builds the frontend and embeds the result in the `woodstar` binary.

`mise run geoip` downloads the current DB-IP City Lite and ASN Lite databases into the gitignored `.local/geoip` directory. `dev-backend`, `test` and `test-postgres` run it automatically and reuse the current month's files.

## Tests

```bash
mise run test
mise run test-postgres
mise run test-e2e
mise run test-all
mise run //web:test
```

`mise run test` runs the dependency-free Go suite. `test-postgres` runs that suite with PostgreSQL component coverage. It and the E2E tasks use the local PostgreSQL service unless `WOODSTAR_TEST_DATABASE_URL` is set.

Run one E2E lifecycle with:

```bash
mise run test-e2e-munki
mise run test-e2e-osquery
mise run test-e2e-santa
mise run test-e2e-mdp
mise run test-e2e-orbit
```

The frontend tests cover navigation permissions, query invalidation and route parameter parsing.

## Lint and format

```bash
mise run lint
mise run fmt-check
mise run format
mise run go-lint
mise run //web:lint
mise run //docs:lint
```

## Generated files

```bash
mise run generate
mise run schema-sync
```

`generate` regenerates `web/openapi.yaml`, the frontend client, and the Go E2E client. `schema-sync` refreshes the vendored osquery schema data.

## Repository checks

```bash
mise run tidy-check
mise run frontend-check
mise run docs
```

`frontend-check` verifies generated API contracts, formatting, lint, tests, types and the production application build. `docs` checks formatting, lint and types, then builds the documentation site.
