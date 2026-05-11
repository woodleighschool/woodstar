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
- `internal/agents/` for the Orbit + osquery substrate (the canonical observation root).
- `internal/santa/` and `internal/munki/` for the optional native capabilities (subtrees materialise when that work begins).
- `internal/api/` for HTTP composition only — routes, middleware, server lifecycle.
- `web/src/` for the React/Vite frontend; `web/public/` for static assets.
- `docs/` for focused engineering notes (working notes, uncommitted).
- `deploy/`, `charts/`, or root-level compose files for deployment artefacts.

Avoid catch-all packages such as `internal/app`, `internal/domain`, `internal/common`, `pkg/utils`. Keep `README.md` concise; detailed engineering notes belong in focused docs.

The full target tree lives in the Architecture Quick Reference at the end of this file.

## Ownership Rules

1. **Orbit/osquery is the canonical observation root.** Only `internal/agents/` creates hosts. Santa and Munki ingest paths look up existing hosts and no-op when absent; they never insert into `hosts`.

2. **`internal/software/` is observed inventory.** Munki manifests are desired state and live in `internal/munki/`. osquery seeing Munki installs via `munki_installs` is still observation and writes to `software/`. No desired-state engine in `software/`.

3. **Santa and Munki are optional native capabilities.** Woodstar runs with only Orbit/osquery. Santa enriches host / security / rules / events views. Munki enriches software / package / install-state views. The core stays coherent if either or both are absent.

4. **`scope/` stays concrete.** Labels are the main targeting primitive. No generic targeting-expression engine. When Santa needs its own scoping shape it gets a parallel type next door, not a generalisation.

5. **`secrets/` stays Orbit/osquery-only.** No pre-shaped `kind` discriminator for Santa/Munki. Santa sync tokens and Munki repo tokens have different protocol lifecycles and get their own packages when they ship.

6. **Domain types are real.** Each domain owns a `model.go` with an explicit struct; `store.go` maps `sqlc.X → X`. Do not embed `sqlc.X` in domain types.

## Package Dependency Direction

- `auth` may depend on `users`. `users` must not depend on `auth`.
- `agents/ingest` writes observed `hosts` / `software` / `labels` membership.
- `labels` must not import `agents`, `santa`, or `munki`.
- `santa` and `munki` may import `hosts`, `labels`, `scope`, `software`. Never the reverse.
- `internal/api/` composes from capability `api/` subpackages. Capability `api/` packages do not import each other.
- `dbutil`, `database`, `config`, `buildinfo`, `logging`, `platform` are leaves: stdlib + third-party only.
- Cross-capability host enrichment: `hosts` defines an enricher interface; each capability registers an implementation at wiring time. `hosts` never imports `agents` / `santa` / `munki`.

## Build, Test, And Development Commands

These targets do not exist yet. Prefer creating this shape when scaffolding the repo:

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
make test               # go test -race -count=1 -v ./...
make test-openapi       # Validate OpenAPI after API contract changes

# Linting
make lint               # Changed files or practical fast lint target

# Pre-commit
make precommit          # fmt + lint + targeted checks

# Formatting
make fmt                # Go and frontend formatting for touched files
```

Until these targets exist, use direct tool equivalents. If a command becomes routine, add a Make target rather than teaching the repo one-off commands.

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

1. During implementation, run targeted tests and fast linting for touched areas.
2. Before handoff, run the closest available pre-commit/build checks.
3. Fix lint issues in touched files. Avoid repo-wide formatting sweeps unless explicitly requested.

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

When running Go tests, use:

```bash
go test -race -count=1 ./...
```

Use targeted tests during iteration, then broader checks before handoff when practical.

For protocol-facing code, test the actual request/response behavior, not just internal helpers.

When adding Go tests that create files with `os.WriteFile`, use `0o600` or tighter permissions unless the test explicitly needs broader mode bits.

Frontend tests should be added when frontend test tooling exists and the behavior is worth protecting. Do not add a whole web test stack just for one trivial component unless asked.

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
- SQL queries should live in a predictable query directory once database tooling exists.
- Generated database code must not be edited manually.
- API contract changes must update generated or checked-in API documentation once that system exists.
- Prefer Postgres-native, readable SQL over abstractions that hide important behavior.
- Keep dynamic SQL carefully bounded to list filtering, sorting, search, and targeting.

## Architecture Quick Reference

Target tree. The architecture-reset spec at `docs/superpowers/specs/2026-05-11-architecture-reset-design.md` drives the moves; this section is the authoritative summary of the destination.

```text
cmd/
  woodstar/
    main.go            single pane of dependency glass

internal/
  config/  buildinfo/  logging/  platform/  web/
  database/            DB connection, pool, migrations, sqlc gen, dbtest
  dbutil/              pagination, sentinel errors, pgx helpers

  # shared core
  auth/                sessions, login, OIDC, password verification
  users/               local Woodstar accounts, roles, password hashes
  hosts/               canonical host identity + host detail loader
  software/            observed software inventory: titles, versions, paths
  labels/              label entity + store
  scope/               concrete scope joins (LabelScope today)
  secrets/             Orbit/osquery enrollment secrets

  # canonical agent substrate (Orbit + osquery)
  agents/
    store.go             enroll-secret + node-key + agent-identity helpers
    service.go           enrollment / config coordination shared by orbit + osquery
    orbit/               Orbit wrapper endpoints + compatibility stubs
    osquery/             osquery TLS API endpoints
    catalog/             osquery query catalog
    queries/             saved queries + query reports
    checks/              boolean query-backed checks
    livequery/           live-query hub + manager merged
    ingest/              inventory projection + label-membership evaluation
    api/                 admin/frontend handlers for the agent capability

  # future Santa capability (skeleton lands when Santa work begins)
  santa/
    sync/  rules/  events/  configurations/  bundles/  ingest/  api/

  # future Munki capability (skeleton lands when Munki work begins)
  munki/
    repo/  manifests/  catalogs/  packages/
    storage/  cache/  pipeline/  importer/
    reports/  ingest/  api/

  # transport composition
  api/
    routes.go            composes routes from agents/api + santa/api + munki/api + shared
    middleware/
    server.go            *http.Server lifecycle

web/                     React/Vite frontend
```

Update this section when the destination shape itself changes — not every time an intermediate refactor lands.
