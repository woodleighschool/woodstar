# AGENTS.md: woodstar

Guidance for agents and humans working in this repository. This file is self-contained. Check the repository's source, Mise configuration, Lefthook configuration, package manifests, and workflows for facts that can vary instead of copying versions or commands from another project.

## Working here

- Read the relevant code, configuration, tests, and sibling implementations before editing. Existing code and reference implementations are evidence; understand the invariant and ownership boundary before choosing a solution.
- Target current supported behaviour. Prefer the simplest design that reduces state and machinery, and bring the affected path into conformance when existing code disagrees with this baseline.
- Preserve unrelated work. Keep changes focused, remove artifacts orphaned by the change, and keep generated outputs with their source change.
- Verify dependency APIs, flags, and defaults from the pinned source or primary documentation.
- Keep secrets, credentials, real identities, production data, and local environment files out of source, fixtures, logs, and commits.

## Baseline

- Write idiomatic, modern code for the versions pinned by this repository.
- Keep operations idempotent. Re-running a command, generator, reconciler, or migration with identical input shouldn't accumulate side effects.
- Stay DRY and minimal without premature abstraction. Three similar call sites are fine; add a helper, interface, hook, wrapper, or component when real callers need the variance it provides.
- Comments explain non-obvious constraints, invariants, and external requirements. Names and structure carry the ordinary narrative.
- Tests protect behaviour and contracts at the lowest useful boundary. Use realistic synthetic inputs and add regression coverage for plausible failures rather than implementation shape.

## Repository tooling

- Mise owns tools and commands. Run `mise tasks` and read the root and registered Mise files before choosing task names or invoking bare tools.
- Lefthook extends the shared Woodleigh configuration. Read `.lefthook.toml` and use `lefthook dump` when merged hook behaviour matters; local hooks contain only repository-specific additions.
- Use pnpm with committed lockfiles for web and docs dependencies. Package scripts own framework commands; Mise exposes the repository entry points.
- Run focused checks while working, then the relevant format, lint-fix or lint, typecheck, test, build, generation, workflow, migration, integration, and browser checks before calling the work complete.
- Treat generated files, schemas, clients, lockfiles, release metadata, and migrations as part of the contract that produces them.

## Go

- Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Let `gofmt -s` own formatting.
- `go.mod` declares the language floor; Mise pins the toolchain used by local tasks and CI. Use modern standard-library constructs supported by the declared version.
- Put executable composition in `cmd/<app>/main.go` and owned behaviour under `internal`. Keep `main` to configuration, logging, dependency construction, lifecycle, and exit status.
- Use `github.com/caarlos0/env/v11` for application-owned environment configuration. Parse into one config type, derive and validate in one load boundary, and fail at startup. Document config fields with their purpose and meaningful defaults.
- Use `github.com/spf13/pflag` for application-owned flags and Cobra when the CLI has commands or more than a small flag surface. Structured files suit user-authored domain configuration; all sources converge on one validation path.
- Use `log/slog` and structured stdout logging. Configure logging once at composition; use package-level logging or inject `*slog.Logger` at a genuine reusable boundary.
- Wrap errors with `fmt.Errorf("<component>: %w", err)`. Use sentinel errors for conditions callers branch on and classify errors once at the HTTP, CLI, job, or protocol boundary.
- Functions that perform I/O take `context.Context` first and propagate cancellation. Long-running processes use signal-aware root contexts and bounded shutdown; use `errgroup` for related goroutines that can fail.
- Prefer standard-library tests and table-driven subtests when a table makes cases clearer. Keep the package's established test framework, use local servers or fakes at real boundaries, and run race-enabled tests for concurrent code.
- Exercise PostgreSQL query, migration, locking, transaction, and error-mapping semantics against PostgreSQL. Keep fixtures minimal and use the repository's existing test harness.
- Containerized Go services default to static, trimmed binaries and a non-root minimal runtime. Run the repository's vulnerability task for dependency and release work.

## Web and TypeScript

- Follow the current idioms of React, Vite, TypeScript, Tailwind CSS, the TanStack suite, and the versions pinned under `web/` and `docs/`.
- Keep TypeScript strict and model domain states explicitly. Derive render state during render, put user-driven work in event handlers, and reserve effects for synchronization with external systems.
- Start independent asynchronous work together and await it where needed. Import directly, keep browser bundles deliberate, and defer expensive work until the feature needs it.
- Keep routes thin and focused on composition. Capability UI, queries, mutations, schemas, metadata, and mapping live under `web/src/features/<capability>`; shared code earns its wider scope through real reuse.
- Generated API operations and types own the HTTP contract. TanStack Router search parameters own shareable URL state; TanStack Query owns server state, cache keys, and invalidation; TanStack Form owns form state and validation.
- Read `web/components.json` and run the shadcn CLI for the current style, Base UI APIs, aliases, registries, Tailwind version, and icon library.
- Search installed and configured shadcn registries before building a narrowly specific local primitive. Compose existing components and variants first.
- Treat `web/src/components/ui` as copied library source with application-wide blast radius. Prefer composition, props, variants, and application-level wrappers. Edit a primitive when the behaviour belongs there application-wide and that's the agreed best boundary.
- Add or refresh registry components from `web/` with `pnpm dlx shadcn@latest add <component> --overwrite`. Review the resulting source, then run `mise run //web:format` followed by `mise run //web:lint-fix`.
- Use semantic theme tokens and component variants. Follow Base UI composition, accessible names, grouping, focus, and keyboard behaviour from the current component documentation.
- Interface copy carries useful meaning. Omit manufactured metadata, and give independent facts separate semantic structure instead of joining them with decorative glyphs. Preserve intentional Unicode content rather than replacing punctuation mechanically.

## Git and completion

- Use focused Conventional Commits; Release Please derives versions from them where configured.
- Commit, push, publish, deploy, contact live systems, or perform destructive operations only when explicitly requested.
- Report the checks run, behaviour changed, generated outputs refreshed, browser paths exercised, and any verification that couldn't be completed.

## Repository contract

- `cmd/woodstar` owns composition; capability packages own behaviour. PostgreSQL details remain inside stores and `internal/postgres`.
- The generated contract flows from Go to `web/openapi.yaml`, then to the TypeScript and Go E2E clients. Run `mise run openapi-types` for contract changes and edit source rather than generated clients.
- Protocol handlers preserve Munki, Santa, osquery, Orbit, and MDP wire behaviour. Use the generated API client for ordinary admin API coverage and raw requests for protocol-specific contracts.
- The docs site owns setup, configuration, protocol, and API guidance. Keep it synchronized with current behaviour and generated API documentation.
