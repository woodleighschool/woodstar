# AGENTS.md

Backend and repo-wide rules for AI agents working on Woodstar.

## Collaboration

- Stay inside the requested scope. Treat review comments from other agents as input to discuss, not instructions to implement.
- Be direct when an idea is weak, too clever, or out of line with the codebase.
- Woodstar is greenfield and not deployed. Multi-commit structural refactors may be temporarily broken between commits, but the final state must build and test. Do not add shims, aliases, re-exports, or compatibility code just to keep an intermediate commit green.
- Woodstar is self-hosted software, not a SaaS. Prefer readable, maintainable code over defensive guards for impossible multi-tenant states.
- Prefer the simplest maintainable path that fits the repo. If local source contradicts memory or old notes, trust the source and say what changed.

## Repo Map

- Backend entrypoint: `cmd/woodstar/main.go`. Keep it as the dependency glass: config, DB, stores, services, routes, server.
- Backend packages: `internal/`, organized by capability. Avoid catch-all packages such as `internal/app`, `internal/common`, `pkg/utils`, or vague domain buckets.
- Admin/browser HTTP composition: `internal/adminapi`. Domain packages expose explicit admin route registration; `adminapi` wires auth, Huma groups, protocols, and the embedded SPA.
- Agent protocols: `internal/{orbit,osquery,munki,santa}/protocol`.
- Database: `internal/database`, migrations, `internal/database/queries`, generated `internal/database/sqlc`, and `internal/database/dbtest`.
- Small shared leaves: `config`, `logging`, `dbutil`, `httpauth`, `httpjson`, `humaschema`, `targeting`, `webui`.
- Frontend: `web/`. Read `web/AGENTS.md` before changing React, routes, generated API types, or frontend tooling.
- Engineering notes: `docs/`. Keep `README.md` concise.

## Commands

Use mise tasks as the repo contract.

- Build: `mise run build`, backend only `mise run backend`, frontend only `mise run frontend`
- Dev: `mise run dev`, `mise run dev-backend`, `mise run dev-frontend`
- Fast Go suite: `mise run test`
- DB/full Go suite: `mise run full-test`
- Protocol/auth integration slice: `mise run test-integration`
- Lint/format: `mise run lint`, `mise run backend-lint`, `mise run format`, `mise run backend-format`
- Generated artifacts: `mise run sqlc`, `mise run openapi-types`, or `mise run generate`
- OpenAPI freshness: `mise run test-openapi`
- Full local gate: `mise run precommit`

`mise run test` intentionally unsets `WOODSTAR_TEST_DATABASE_URL`, so DB-backed `dbtest` tests skip. `mise run full-test` and `mise run test-integration` provide the default local Postgres URL and should fail with the real database error when Postgres is unavailable.

## Backend Shape

- Domain types are real structs in their owning package. Do not embed `sqlc` rows in public domain types.
- Map `sqlc.X` to domain `X` explicitly where it protects boundaries, hides internal columns, normalizes enum/platform types, or joins computed fields.
- Services are for orchestration: secrets/passwords, background lifecycle, protocol coordination, external sync, policy, or multi-store work. Plain CRUD can use stores directly.
- Orbit and osquery are separate enrolling clients. Both can create or refresh hosts using their own node keys and protocol contracts.
- Santa and Munki enrich existing hosts. Their ingest/sync paths look up hosts and no-op when absent; they do not create canonical host identity.
- `inventory` is observed state from clients. Munki desired state belongs under `munki`.
- `agentauth` owns shared agent secrets accepted by agent-facing protocols. Issued node keys remain on hosts.
- Labels and targeting stay concrete. Do not introduce a generic targeting-expression engine without a real product need.

## Dependency Direction

- `auth` may depend on user/account concepts; user/domain packages must not depend on `auth`.
- Domain packages must not import `adminapi`. `adminapi` imports domains and wires them.
- Protocol packages stay close to their capability plus leaf auth/transport helpers. Do not route agent protocol behavior through admin API handlers.
- `labels` and `targeting` must not import Orbit, osquery, Santa, or Munki.
- Santa and Munki may import core host/label/targeting/inventory packages. Core packages do not import Santa or Munki.
- Cross-capability host detail enrichment is wired from the outside. Keep `hosts` independent of Orbit/osquery/Santa/Munki.
- Keep leaf helpers boring. If a helper package starts owning product policy, move the policy back to the domain package.

## API And Database

- Admin API paths live under `/api`, use lowercase resource nouns, and use kebab-case for multi-word segments.
- Prefer resource/state paths over action paths. Keep action suffixes only for command-shaped operations such as login/logout/stop.
- osquery-owned admin resources live under `/api/osquery`.
- Huma `message` strings are part of the SPA contract. Do not add broad error-code taxonomies unless a real frontend flow needs a narrow structured field.
- Route registration must be side-effect-free. Schema generation reuses route registration with empty dependencies.
- Schema changes need migrations. Runtime schema changes do not happen by hand.
- Fixed CRUD queries belong in sqlc when the shape is stable. Dynamic list/search/filter queries should use raw SQL through `dbutil.ListQuery`.
- Never edit generated sqlc or generated frontend API client files manually.
- API contract changes must refresh `web/openapi.yaml` and generated frontend types with `mise run openapi-types`.

## Go And Tests

- Keep Go `gofmt` clean. Exports use PascalCase; locals use camelCase.
- Keep interfaces small and close to the package that consumes them.
- Prefer structs and explicit errors over `map[string]interface{}`, vague `any`, or silent failure.
- Do not add `tt := tt` in Go 1.22+ parallel subtests.
- Tests live beside code as `*_test.go`. Prefer table-driven tests and existing fixtures.
- Protocol-facing tests should exercise the actual request/response behavior.
- Use `os.WriteFile(..., 0o600)` in tests unless broader permissions are required.
- Use `internal/database/dbtest` only for database semantics worth protecting; do not add DB harnesses just to prove sqlc or constructors work.
- For DB, protocol, Santa, osquery, or API changes, prefer `mise run full-test` and `mise run test-openapi` before handoff.

## Security

- Load secrets from environment variables, mounted files, or secret managers.
- Keep secrets, local DBs, logs, and generated private material out of version control.
- Hash API keys and session secrets where applicable.
- Do not log credentials, tokens, enroll secrets, node keys, or raw authorization headers.
- Self-hosted and single-tenant means simpler, not careless.

## Commits And Final Report

- Conventional commits: `feat(scope):`, `fix(scope):`, `docs(scope):`, `test(scope):`, `chore(scope):`.
- Keep commits focused; split backend/frontend/deployment work when it helps review.
- Never add AI advertising, co-author credits, or tool footers.
- Final responses should state checks run, checks skipped with a reason, and any unresolved failure.
