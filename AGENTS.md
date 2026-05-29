# AGENTS.md

Repository guidelines for AI coding agents working on Woodstar.

## Owner Collaboration Notes

- Do not implement review-suggested or extra changes outside the requested scope without explicit user approval first.
- Treat other agent/Codex/CodeRabbit feedback as input to discuss, not automatic action.
- Be direct when an idea is weak, overcomplicated, or inconsistent with the existing direction.
- Prefer the simplest maintainable path that fits the repo over clever architecture.
- If local source/reference repos contradict memory, inspect the source and say what changed.

## Working Discipline (Non-Deployed Greenfield)

Woodstar is not deployed, not published, has no upstream, no users. Treat that as a freedom, not an excuse:

- Structural commits do not have to compile or pass tests. A commit's job is to move the tree closer to its target shape; intermediate breakage between structural commits is acceptable and expected.
- No between-commit bandaids. Do not add temporary shims, type aliases, re-exports, or `// TODO: remove after step N` stubs solely to keep the build green between structural steps. They distort the diff and outlive their purpose.
- Verification happens at the end of a multi-commit refactor, not after each commit. Build and tests must pass on the final commit.
- No churn-avoidance. If a thing is in the wrong package, move it now even when the caller-list is long.
- No backwards-compatibility code. There is no caller to be compatible with.

For single-commit changes (bugfix, small feature) the usual "code compiles, touched tests pass" rule applies.

## Project Structure & Module Organization

- `cmd/woodstar/main.go` is the single pane of dependency glass: config load → DB open → stores/services constructed → routes composed → server started. No `internal/app/` package.
- `internal/` holds product packages organised by capability.
- `internal/database/` for DB connection, pool, migrations, sqlc gen, dbtest.
- `internal/dbutil/` for pagination, sentinel errors, pgx helpers.
- `internal/httpjson/` for tiny JSON encode/decode helpers used by raw `net/http` protocol endpoints.
- `internal/agentauth/` for admin-managed shared agent secrets accepted by Orbit/osquery enrollment and Santa sync.
- `internal/orbit/` for Orbit enrollment, Orbit config, and Orbit-specific protocol endpoints.
- `internal/osquery/` for osquery TLS-plugin enrollment, config, distributed query/log handling, catalog, checks, reports, live queries, and inventory projection. Orbit can provide extension tables that osquery queries use, but osquery remains the protocol/client boundary for those queries.
- `internal/santa/` and `internal/munki/` for the optional native capabilities (subtrees materialise when that work begins).
- `internal/api/` owns the **admin HTTP surface**: Huma route registration, handlers, middleware, server lifecycle. Every admin (session-authed, JSON, OpenAPI-documented) endpoint lives in `internal/api/handlers/`, regardless of which domain package owns the entity. Capability packages register handlers here; they do not host them.
- `internal/{capability}/protocol/` owns **agent-facing HTTP endpoints** for that capability (chi-routed, node-key auth, protocol-shaped). Today: `internal/orbit/protocol/` registers Orbit endpoints and `internal/osquery/protocol/` registers osquery TLS-plugin endpoints. When Santa or Munki ship their own agent protocols, they get `internal/santa/protocol/` / `internal/munki/protocol/`.
- `web/src/` for the React/Vite frontend; `web/public/` for static assets.
- `docs/` for focused engineering notes (working notes, uncommitted).
- `deploy/`, `charts/`, or root-level compose files for deployment artefacts.

Avoid catch-all packages such as `internal/app`, `internal/domain`, `internal/common`, `pkg/utils`. Keep `README.md` concise; detailed engineering notes belong in focused docs.

The full target tree lives in the Architecture Quick Reference at the end of this file.

## Ownership Rules

1. **Orbit and osquery are separate enrolling clients.** Both can create or refresh hosts through `hosts` using their own node-key columns and protocol contracts. Orbit can provide extension tables that osquery inventory queries use, but missing extension tables are still osquery query result statuses/messages, not a separate server-side runtime mode. Santa and Munki ingest paths look up existing hosts and no-op when absent; they never insert into `hosts`.

2. **`internal/software/` is observed inventory.** Munki manifests are desired state and live in `internal/munki/`. osquery seeing Munki installs via `munki_installs` is still observation and writes to `software/`. No desired-state engine in `software/`.

3. **Santa and Munki are optional native capabilities.** Woodstar runs with only Orbit/osquery. Santa enriches host / security / rules / events views. Munki enriches software / package / install-state views. The core stays coherent if either or both are absent.

4. **`scope/` stays concrete.** Labels are the main targeting primitive. No generic targeting-expression engine. When Santa needs its own scoping shape it gets a parallel type next door, not a generalisation.

5. **`agentauth/` owns shared agent secrets.** Agent secrets are the admin-managed shared credentials accepted by agent-facing protocols. `agent=orbit` is accepted by Orbit and osquery enrollment; `agent=santa` is accepted as the Santa sync bearer credential. Do not add Munki until its server exists. Orbit/osquery issued node keys remain on `hosts`.

6. **Domain types are real.** Each domain owns a `model.go` with an explicit struct; `store.go` maps `sqlc.X → X`. Do not embed `sqlc.X` in domain types. Keep explicit mappers where they enforce domain/public DTO boundaries, hide internal columns, normalize enum/platform types, or join computed fields. Only introduce generated mapping when the source and destination are truly mechanical mirrors across enough call sites to justify owning the generator; do not use reflection/mapstructure-style runtime mappers.

7. **Services are for orchestration, not symmetry.** Do not create a `Service` just because a domain has a store. Keep a service when it owns policy, password/secret handling, background lifecycle, protocol coordination, external API sync, or multi-store orchestration. Plain CRUD domains can expose stores directly to the admin handler layer.

## Package Dependency Direction

- `auth` may depend on `users`. `users` must not depend on `auth`.
- `osquery/ingest` writes observed `hosts` / `software` / `labels` membership.
- `labels` must not import `orbit`, `osquery`, `santa`, or `munki`.
- `santa` and `munki` may import `hosts`, `labels`, `scope`, `software`. Never the reverse.
- `internal/api/handlers/` is the single home for admin Huma handlers. It may import any domain package (`hosts`, `labels`, `osquery/reports`, `osquery/checks`, `osquery/livequery`, etc.). Domain packages never import `handlers`.
- `internal/{capability}/protocol/` packages are leaves of the protocol surface: their imports stay inside their own capability subtree (e.g. `orbit/protocol` may import `orbit`; `osquery/protocol` may import `osquery`), plus leaf packages such as `agentauth` and `enrollment`.
- Route-shape rule for new endpoints: session-authed REST/JSON → `internal/api/handlers/`; agent-authed protocol (Orbit/osquery TLS plugin/etc.) → `internal/{capability}/protocol/`. Do not split admin handlers by domain ownership.
- `dbutil`, `database`, `config`, `buildinfo`, `logging`, `scope` are leaves: stdlib + third-party only.
- Cross-capability host enrichment: `hosts` defines an enricher interface; each capability registers an implementation at wiring time. `hosts` never imports `orbit` / `osquery` / `santa` / `munki`.
- Keep `dbutil` as the small shared database-helper leaf until a split removes real import pressure. It may own list/WHERE builders, sentinel persistence errors, and pgx/Postgres helpers, but it must not become a generic application `common` package.

## Admin API Naming

- Admin API paths live under `/api`.
- Use lowercase resource nouns for path segments. Use kebab-case when a segment has multiple words (`/api/live-queries`, `/api/agent-secrets`, `/api/account/api-key`).
- Use plural collection nouns for ordinary collections (`/api/hosts`, `/api/users`, `/api/osquery/reports`). Singular singleton resources are acceptable when there is only one resource in the caller's context (`/api/account`, `/api/auth/session`).
- osquery-owned admin resources live under `/api/osquery` (`/api/osquery/reports`, `/api/osquery/checks`) rather than at the API root.
- Prefer state/resource paths over action paths. Keep action suffixes only for command-shaped operations that do not naturally address a separate resource (`/api/auth/login`, `/api/auth/logout`, `/api/setup`, `/api/live-queries/{id}/stop`).
- Treat the admin API as the contract for Woodstar's own SPA and small trusted tooling, not as a future public SaaS API. Keep OpenAPI/type generation where it protects the SPA contract, but do not add public-API ceremony (extra tags, summaries, compatibility envelopes, broad security-scheme noise, or speculative error taxonomies) unless a real frontend, CLI, or protocol caller needs it.
- Huma error `message` strings are part of the SPA contract today. Do not add a parallel `error_code` taxonomy speculatively; if a frontend flow needs programmatic branching, add the narrow structured field with that flow and update the generated client/types in the same slice.
- Keep Huma route registration explicit per resource. Share boring mechanics such as paginated envelopes, bulk-ID parsing, list builders, and sentinel error mapping, but do not introduce a generic CRUD/router framework for handlers that need resource-specific request bodies, summaries, auth, or response mapping. `//nolint:dupl` on paired resources such as reports/checks is acceptable when the duplication preserves clear API contracts.
- Huma route registration must stay side-effect-free: registering routes may capture services/stores but must not call them. `BuildSchemaAPI` reuses the admin route registration with empty dependencies for schema generation; if that ever becomes unsafe, split schema-only registration deliberately instead of adding runtime shims.

## Build, Test, And Development Commands

Use the Makefile targets as the repo contract. Prefer these repo-wide commands over bespoke shell commands or per-file lint/format runs unless you are narrowing a specific failure.

```bash
# Build
make build              # Frontend bundle + Go binary
make backend            # Go binary only
make frontend           # Frontend bundle only

# Development
make dev                # Backend + frontend development loop
make dev-backend        # Backend only
make dev-frontend       # Frontend only

# Testing
make test               # Fast Go suite; DB tests skip because WOODSTAR_TEST_DATABASE_URL is unset
make full-test          # Full/e2e Go suite against real Postgres via WOODSTAR_TEST_DATABASE_URL
make test-integration   # DB-backed protocol/auth integration slice
make test-openapi       # Validate OpenAPI after API contract changes

# Linting
make lint               # Backend + frontend lint
make backend-lint       # golangci-lint run + deadcode -test ./...
make frontend-lint      # pnpm run lint

# Formatting
make format             # Frontend + backend formatting
make backend-format     # golangci-lint fmt
make frontend-format    # pnpm run format
make fmt                # Alias for make format

# Pre-commit
make precommit          # format + lint + full-test + OpenAPI check
```

`WOODSTAR_TEST_DATABASE_URL` is intentionally implied by `make full-test` and `make test-integration`. If Postgres is not reachable at that URL, the test should fail with the real Go/database error. Empty or unset `WOODSTAR_TEST_DATABASE_URL` means DB-backed `dbtest` tests skip.

If a command becomes routine, add a Make target rather than teaching the repo one-off commands.

## Linting Strategy

Use strict linting to catch common AI-generated slop early. Keep the configuration practical, but start from these expectations:

| Linter | Purpose | Threshold |
|--------|---------|-----------|
| dupl | Catch code duplication | 100 tokens |
| gocognit | Cognitive complexity | 15 |
| funlen | Function length | 80 lines |
| interfacebloat | Interface size | 5 methods |
| errcheck | Unchecked errors | All, including type assertions |
| gocritic | Non-idiomatic patterns | diagnostic + style + performance |

Workflow:

1. During implementation, targeted tests are fine for diagnosis, but use `make format` and `make lint` for the lint/format pass.
2. Before handoff, run `make precommit` when practical. For database, protocol, Santa, osquery, or API changes, strongly prefer `make full-test` over the lightweight `make test`.
3. Fix lint issues in touched files. If repo-wide lint exposes unrelated inflight work, do not paper over it; report the blocker and keep the requested change focused.

## Coding Style & Naming Conventions

Go:

- Keep code `gofmt`-clean.
- Use PascalCase for exports and camelCase for locals.
- Keep interfaces small and local to the package that consumes them.
- Prefer explicit structs over `map[string]interface{}` or vague `any` plumbing.
- Prefer explicit error handling over silent failures.
- Do not use `tt := tt` in Go 1.22+ parallel subtests.

Frontend:

- Use React with TypeScript.
- Organize modules by feature.
- Keep reusable UI primitives separate from product-specific components.
- Prefer clear names over abbreviations.
- Keep table, form, and route code typed.

General:

- Prefer established libraries for routing, migrations, database access, OpenAPI, OIDC, S3-compatible storage, and frontend table/form/query state.
- Do not reimplement infrastructure for sport.
- Avoid premature abstraction. Add a seam when there is real duplication, transaction pressure, test pressure, or protocol complexity.
- Woodstar is self-hosted and single-tenant. Do not write paranoid multi-tenant SaaS code for impossible states.

## Code Shape

- Prefer behavior-bearing branches only.
- If multiple `switch` cases return the same value as `default`, collapse them.
- In boolean classifiers, list exceptional cases and let the common path be default.
- Do not add documentation-only branches unless they enforce something mechanically via compiler, linter, or tests.
- Keep comments short and useful. Explain why something is surprising, not what obvious code does.
- Do not leave "temporary" hacks in mainline code just to satisfy a build.

## React Effects

- Use `useEffect` only to sync with external systems such as DOM APIs, subscriptions, timers, or network behavior.
- Avoid derived state in Effects; calculate during render, or use `useMemo` for expensive computation.
- Put user-driven logic in event handlers, not Effects.
- To reset state, prefer a `key` or render-time adjustment instead of an Effect.
- Fetch Effects must guard against stale responses with cleanup or abort behavior.

## Testing Guidelines

Backend tests should live beside implementations as `*_test.go`. Prefer table-driven tests.

When running Go tests, prefer the Make targets:

```bash
make test       # Fast suite without DB-backed tests
make full-test  # Full/e2e suite with real Postgres
```

Use targeted `go test` commands during iteration when they shorten the loop, then run the broadest relevant Make target before handoff. For DB-backed behavior, protocol surfaces, and Santa/osquery ingestion, that usually means `make full-test`.

For protocol-facing code, test the actual request/response behavior, not just internal helpers.

When adding Go tests that create files with `os.WriteFile`, use `0o600` or tighter permissions unless the test explicitly needs broader mode bits.

Frontend tests should be added when frontend test tooling exists and the behavior is worth protecting. Do not add a whole web test stack just for one trivial component unless asked.

`internal/database/dbtest` is for store/database semantics that are worth protecting with a real Postgres schema. Do not add a dbtest harness just because a package has a store; prefer service/API coverage when that is the meaningful contract, and skip redundant store tests when they would only prove sqlc or a constructor works.

## Commit & Pull Request Guidelines

Follow conventional commit style when making commits:

- `feat(scope): ...`
- `fix(scope): ...`
- `docs(scope): ...`
- `test(scope): ...`
- `chore(scope): ...`

Keep commits focused. Split backend, frontend, and deployment changes when that makes review easier.

Never add:

- AI-generated advertising.
- AI co-author credits.
- Tool-specific generation footers.

PRs should include a concise summary, testing notes, and screenshots for visible UI changes.

## Verification Checklist

Before handoff:

1. Changed files are formatted.
2. Targeted tests for touched packages pass.
3. Build or typecheck succeeds for touched surfaces where practical.
4. API changes include matching OpenAPI/schema updates once that system exists.
5. Database changes include migrations and query/model updates.
6. Agent/protocol-facing code is checked against upstream client behaviour, not memory.

For multi-commit structural refactors (see **Working Discipline**), this checklist applies to the **final** commit, not each intermediate one.

## Security & Configuration

- Load secrets from environment variables, mounted files, or secret managers.
- Keep secrets, local databases, logs, and generated private material out of version control.
- Hash API keys and session secrets where applicable.
- Do not log credentials, tokens, enroll secrets, node keys, or raw authorization headers.
- Runtime schema changes must go through migrations, not manual database edits.
- Treat self-hosted as simpler operationally, not as permission to be careless with credentials.

## API & Database Change Rules

- Database schema changes must ship as migrations.
- SQL queries live in `internal/database/queries/*.sql`; sqlc-generated code lives in `internal/database/sqlc/`.
- Generated database code must not be edited manually.
- API contract changes must update generated or checked-in API documentation once that system exists.
- Prefer Postgres-native, readable SQL over abstractions that hide important behavior.

### Picking sqlc vs raw SQL

- **Fixed CRUD on a single table** (insert with known columns, update by id, delete by id, lookup by id) → sqlc. The generated functions are concrete and discoverable.
- **List, search, filter, and any other query whose WHERE/ORDER varies at runtime** → raw SQL via `dbutil.ListQuery`. sqlc can't model dynamic WHERE cleanly.
- **Shared base SELECTs** consumed by both fixed and dynamic queries may stay as raw `const xxxSelectSQL` until they cause friction.

Inconsistent today: query/check Create/Update/Delete use raw SQL where sqlc would fit. Acceptable as-is; migrate if/when a third caller emerges that would benefit.

## Architecture Quick Reference

Target tree. The architecture-reset spec at `docs/superpowers/specs/2026-05-11-architecture-reset-design.md` drives the moves; this section is the authoritative summary of the destination.

```text
cmd/
  woodstar/
    main.go            single pane of dependency glass

internal/
  config/  buildinfo/  logging/  web/
  database/            DB connection, pool, migrations, sqlc gen, dbtest
  dbutil/              pagination, sentinel errors, pgx helpers
  httpjson/            JSON transport helpers for raw net/http endpoints

  # shared core
  auth/                sessions, login, OIDC, password verification
  users/               local Woodstar accounts, roles, password hashes
  hosts/               canonical host identity + host detail loader
  software/            observed software inventory: titles, versions, paths
  labels/              label entity + store
  scope/               concrete targeting primitives (LabelScope, Platform)
  agentauth/           shared agent-secret store and bearer helpers

  orbit/
    service.go           Orbit service (config + enrollment business logic)
    protocol.go          Orbit wire DTOs
    protocol/            Orbit agent-facing HTTP endpoints

  osquery/
    service.go           osquery service (config + enrollment + dispatch)
    protocol.go          osquery wire DTOs
    protocol/            osquery TLS-plugin endpoints
    queries/             saved osquery SQL definition normalization
    catalog/             osquery query catalog
    reports/             saved scheduled reports + per-host result snapshots
    checks/              boolean query-backed checks (domain + store)
    livequery/           live-query hub + manager
    ingest/              inventory projection + label-membership evaluation

  # future Santa capability (skeleton lands when Santa work begins)
  santa/
    sync/  rules/  events/  configurations/  bundles/  ingest/
    protocol/            Santa sync protocol endpoints (when applicable)

  # future Munki capability (skeleton lands when Munki work begins)
  munki/
    repo/  manifests/  catalogs/  packages/
    storage/  cache/  pipeline/  importer/
    reports/  ingest/
    protocol/            Munki repo HTTP endpoints (when applicable)

  # admin HTTP surface — every admin Huma route, regardless of which
  # domain package owns the entity
  api/
    handlers/            Huma route registration per resource
                         (hosts, labels, users, software, agent_secrets,
                          reports, checks, live_queries, auth, …)
    middleware/
    routes.go            wires handlers onto the chi router
    server.go            *http.Server lifecycle

web/                     React/Vite frontend
```

Update this section when the destination shape itself changes — not every time an intermediate refactor lands.
