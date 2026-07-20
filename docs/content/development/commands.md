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
docker compose up -d postgres
```

`mise run deps` downloads Go modules and installs frontend dependencies in `web/`. Compose provides the PostgreSQL service shared by local development and dependency-bearing tests. Woodstar runs through mise; S3 integration starts its own ephemeral Garage dependency.

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
mise run test-postgres
mise run test-integration-storage
mise run test-integration
mise run test-e2e-munki
mise run test-e2e-osquery
mise run test-e2e-santa
mise run test-e2e-mdp
mise run test-e2e-orbit
mise run test-e2e
mise run test-all
```

`mise run test` runs `go vet ./...` and the dependency-free Go suite with race detection. It needs neither PostgreSQL nor Docker. Pure validation, mapping, protocol error handling, local file storage, and service behavior with Woodstar-owned fakes belong here.

`mise run test-postgres` selects `//go:build postgres` component tests under `internal/`. Each test creates, migrates, and drops an isolated database on `WOODSTAR_TEST_DATABASE_URL`. The task defaults that URL to the checked-in Compose PostgreSQL service, but it never starts Compose itself.

`mise run test-integration-storage` selects `//go:build integration` and runs the S3 contract against an ephemeral Garage testcontainer. The dependency-free file implementation runs in the normal suite. `mise run test-integration` is the aggregate provider lane.

The E2E tasks select `//go:build e2e`, compile a real Woodstar server, and exercise application lifecycles. The Orbit lane replays representative Orbit and osquery requests from checked-in protocol fixtures. Santa uses the same fixture approach for its protobuf requests while retaining the stateful clean-sync, rule-download, event-upload, checkpoint, and normal-sync exchange. Tagged dependency lanes fail when PostgreSQL or Docker is unavailable; they never turn a requested test into a skip. `mise run test-all` runs every lane.

The frontend has no test runner. Its verification is `mise run //web:lint`, `mise run //web:typecheck`, generated OpenAPI clients, and `mise run //web:build`.

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

`generate` regenerates the OpenAPI schema, frontend client, and Go E2E client.

## Complete Test Gate

```bash
docker compose up -d postgres
mise run test-all
```

The aggregate test gate is intentionally explicit about its dependencies. Use `mise run fmt-check`, `mise run tidy-check`, `mise run lint`, and `mise run build` as separate repository checks.
