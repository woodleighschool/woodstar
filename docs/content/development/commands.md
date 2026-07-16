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
mise run test-integration-munki
mise run test-integration-osquery
mise run test-integration-santa
mise run test-integration-orbit
mise run test-integration
mise run test-openapi
```

`mise run test` is the focused Go suite. It uses a real PostgreSQL database with race detection and fresh test results. The target integration tasks run one compiled-server lifecycle; `mise run test-integration` runs all four lifecycles in one integration process. Every test task supplies the default local PostgreSQL URL when `WOODSTAR_TEST_DATABASE_URL` is unset.

Munki, osquery, and Santa fail when their prerequisites or assertions fail. Orbit may skip locally only when Docker, Orbit, or osqueryd is absent. CI requires those Orbit prerequisites, so the same condition fails there.

The frontend has no test runner. Its verification is `mise //web:lint`, `mise //web:typecheck`, `mise run test-openapi`, and `mise //web:build`.

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

The precommit task runs format, lint, tidy, build, the focused PostgreSQL-backed suite, all compiled integrations, and OpenAPI freshness in that order.
