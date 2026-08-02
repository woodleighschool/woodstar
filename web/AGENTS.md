# AGENTS.md

Frontend guidance for Woodstar.

## Approach

- Stay within the requested scope and preserve unrelated local changes.
- This is a dense self-hosted admin application, not a marketing site or SaaS shell.
- Use existing app chrome, components, tokens, and feature patterns before creating another abstraction.
- Simplify and modernize existing code instead of adding aliases, barrels, compatibility exports, or duplicate state layers.
- Generated API types and current feature ownership are authoritative.

## Stack and Commands

- Stack: React, Vite, TypeScript, Tailwind, TanStack Router/Form/Query/Table, and shadcn-style primitives.
- Source: `web/src`; static assets: `web/public`; generated bundle: `web/dist`.
- Development: `mise run //web:dev`
- Build: `mise run //web:build`
- Lint: `mise run //web:lint`; fixes: `mise run //web:lint-fix`
- Format: `mise run //web:format`; check: `mise run //web:fmt-check`
- Regenerate API client: `mise run openapi-types`

`web/openapi.yaml` and `web/src/lib/api-client/` are generated. Don't edit generated client files by hand.

## Organization

- App composition and router/query setup live in `src/app`; route modules in `src/routes` stay thin.
- Capability UI, query state, mutations, metadata, and mapping live in `src/features/<capability>`.
- Shared cross-capability UI belongs in `src/components`; design-system primitives stay in `src/components/ui`.
- `src/hooks` and `src/lib` are feature-neutral technical infrastructure. One-feature helpers stay with that feature.
- Use the established `@components`, `@features`, `@hooks`, and `@lib` imports. Don't add a catch-all alias.
- Mutable resources use thin create/edit shells with shared fields. Read-only resources use list/detail shapes.

## Data and UI Rules

- Fetch through generated operations and `unwrap`; query keys and invalidation belong to the owning feature.
- Server-backed collections keep query, filters, sorting, page, and size in validated route search parameters.
- The backend owns API collection search, filtering, sorting, and pagination. React renders the returned projection.
- Forms use TanStack Form, zod schemas, and the existing form-field/action components.
- Use CSS variable tokens and existing primitives. Avoid hardcoded palette values, nested card clutter, and long operational copy.
- Use effects only for external synchronization. Keep user-driven behavior in event handlers and derive render state directly.

## Checks

- Frontend changes normally require format, lint/typecheck, and a production build.
- API changes also require generated-contract checks.
- Check browser-visible changes in a running app when that is in scope; state clearly when browser proof wasn't run.
