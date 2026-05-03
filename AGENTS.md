# AGENTS.md

Repository guidelines for AI coding agents working on Woodstar.

## Owner Collaboration Notes

- Do not implement review-suggested or extra changes outside the requested scope without explicit user approval first.
- Treat other agent/Codex/CodeRabbit feedback as input to discuss, not automatic action.
- Be direct when an idea is weak, overcomplicated, or inconsistent with the existing direction.
- Prefer the simplest maintainable path that fits the repo over clever architecture.
- If local source/reference repos contradict memory, inspect the source and say what changed.

## Project Structure & Module Organization

Expected shape once code exists:

- `cmd/woodstar` for the Go entrypoint.
- `internal/` for product packages and application wiring.
- `internal/db` for migrations, SQL queries, and generated database code.
- `web/src` for the React/Vite frontend.
- `web/public` for frontend static assets.
- `docs/` for focused engineering notes.
- `deploy/`, `charts/`, or root-level compose files for deployment artifacts.

Keep shared helpers small and explicit. Avoid catch-all packages such as `internal/app`, `internal/domain`, `internal/common`, or `pkg/utils` unless a concrete repeated use justifies them.

Keep `README.md` concise. Detailed engineering notes belong in focused docs.

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
- Do not add backward compatibility shims unless explicitly requested.
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
6. Agent/protocol-facing code is checked against upstream client behavior, not memory.

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

Planned high-level shape:

```text
cmd/woodstar/              Go entrypoint
internal/api/              HTTP routing and middleware
internal/auth/             Human auth, sessions, roles
internal/db/               Migrations, queries, generated DB code
internal/hosts/            Host lifecycle, list/detail, labels
internal/agent/            Orbit-managed agent endpoints and protocol glue
internal/inventory/        Inventory projection from agent results
internal/queries/          Saved queries, reports, checks
internal/software/         Software inventory and title model
internal/santa/            Future Santa module
internal/munki/            Future Munki module
web/src/                   React frontend
```

Keep the actual structure honest to the code that exists. Update this section when the scaffold lands.
