# AGENTS.md

Repository guidance for Woodstar.

## Approach

- Stay within the requested scope and preserve unrelated local changes.
- Woodstar is purpose-built, self-hosted internal software, not a SaaS platform. Prefer direct code for demonstrated needs.
- Simplify and modernize existing code before adding abstractions, compatibility layers, re-exports, or generic engines.
- Woodstar is the rich shared tooling baseline, but each capability still owns its application behavior.

## Repository Map

- Process composition and subcommands: `cmd/woodstar`
- Capability-owned backend code: `internal/`
- HTTP server and handlers: `internal/api`
- Database and migrations: `internal/database`
- Protocols: `internal/{orbit,osquery,munki,santa}/protocol`
- Cross-system tests: `test/e2e`; provider/storage integration: `test/integration`
- Frontend: `web/`; read `web/AGENTS.md` before changing it
- Documentation: `docs/`
- Shared schemas and generated inputs: `schema/`

Avoid catch-all packages and vague utility layers. Policy stays with the capability that owns it.

## Commands

Use Mise tasks as the repository contract.

- Dependencies: `mise run deps`
- Build: `mise run build`; backend only: `mise run backend`; web only: `mise run //web:build`
- Tests: `mise run test`; PostgreSQL: `mise run test-postgres`; all lanes: `mise run test-all`
- E2E: `mise run test-e2e` or a focused `mise run test-e2e-{munki,osquery,santa,mdp,orbit}`
- Lint: `mise run lint`; fixes: `mise run lint-fix`
- Format: `mise run format`; check: `mise run fmt-check`
- Generated OpenAPI and clients: `mise run openapi-types`
- Module and workflow checks: `mise run tidy-check`, `mise run workflow-lint`

`mise run test` requires neither PostgreSQL nor Docker. Tagged database and integration lanes require their named services and don't silently skip. Use `//web:*` and `//docs:*` tasks when only one frontend is in scope.

## Backend Rules

- `cmd/woodstar/main.go` owns central composition. Services orchestrate behavior; they aren't plain CRUD wrappers.
- Domain types live in their capability. Keep core host packages independent of Orbit, osquery, Santa, and Munki.
- Orbit and osquery enroll hosts. Santa and Munki enrich existing hosts rather than creating canonical identity.
- Use raw pgx stores with a canonical projection shared by Get and List. Don't add an ORM or sqlc.
- Persist timestamps with SQL `now()` and re-read created or updated records for response bodies.
- API paths use lowercase resource nouns. Route registration remains side-effect-free.
- API changes regenerate `web/openapi.yaml`, frontend clients, and the Go E2E client in the same change.

## Engineering Rules

- Prefer concrete Go types, small consumer-owned interfaces, and explicit wrapped errors.
- Assert behavior at the lowest useful layer. Use PostgreSQL for SQL semantics and don't mock SQL.
- The generated contract chain is Go to `web/openapi.yaml` to frontend, E2E client, and docs.
- Keep secrets, enrollment material, node keys, local databases, and real identities out of logs and version control.
- Shared hooks stay fast and staged-file focused. Builds, generation, and integration lifecycles belong in Mise tasks and CI.

## Commits

- Use focused Conventional Commits.
- Don't push, deploy, publish, release, or contact live systems unless explicitly requested.
- Report checks run, skipped checks, proof boundaries, and unresolved failures.
