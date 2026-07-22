---
sidebar_position: 2
title: Commands
description: Build, test, lint, format, and generate Woodstar.
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
mise run //web:build
```

`mise run build` builds the frontend and embeds the result in the `woodstar` binary.

## Tests

```bash
mise run test
mise run test-postgres
mise run test-integration
mise run test-e2e
mise run test-all
```

`mise run test` runs the dependency-free Go suite. `test-postgres` and the E2E tasks use the local PostgreSQL service unless `WOODSTAR_TEST_DATABASE_URL` is set. The integration lane starts its own S3 test service through testcontainers.

Run one E2E lifecycle with:

```bash
mise run test-e2e-munki
mise run test-e2e-osquery
mise run test-e2e-santa
mise run test-e2e-mdp
mise run test-e2e-orbit
```

The frontend has no test suite. Use its lint, typecheck, generated client, and production build checks.

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
mise run openapi-types
mise run generate
mise run schema-sync
```

`openapi-types` regenerates `web/openapi.yaml`, the frontend client, and the Go E2E client. `schema-sync` refreshes the vendored osquery schema data.

## Repository checks

```bash
mise run tidy-check
mise run workflow-lint
mise run //docs:build
```
