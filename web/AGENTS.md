# AGENTS.md

Frontend rules for work under `web/`.

## Stack / Commands

- React 19, Vite, TypeScript, Tailwind v4, TanStack Router/Form/Query/Table, shadcn-style primitives, lucide icons.
- Source: `web/src`; static assets: `web/public`; production bundle: `web/dist`.
- From repo root: `mise run //web:dev`, `mise run //web:build`, `mise run //web:lint`, `mise run //web:format`, `mise run openapi-types`.
- Inside `web/`: `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm format`, `pnpm openapi:types`.
- `web/openapi.yaml` and `web/src/lib/api-client/` are generated. Do not edit generated client files by hand.

## Product / UI

- This is a self-hosted admin SPA served by the Go backend. No marketing pages, onboarding filler, SaaS ceremony, or speculative multi-tenant UI.
- Use established app chrome: `PageShell`, `PageHeader`, sidebar, breadcrumbs, and existing layout components.
- Use CSS variable tokens and existing UI primitives. Avoid hardcoded hex colors or raw palette utilities.
- Use lucide icons for common actions. Keep operational copy short.
- Match the existing dense admin feel. Do not nest cards inside cards or turn page sections into floating cards.

## Organization

- Application composition, router/query setup, and app-level fallbacks live directly in `src`;
  route declarations stay thin in `src/routes`.
- Business UI, query keys/options, hooks, mutations, metadata, and mapping logic live together in
  `src/features/<capability>`. Use resource subdirectories inside a broad capability such as Munki,
  Santa, osquery, or Directory.
- Feature query modules own their key factories, loader-compatible query options, hooks, mutations,
  and invalidation. Do not rebuild a global query-key registry or a parallel `lib/queries` layer.
- `src/components` is for UI used by multiple capabilities. Keep design-system source in
  `components/ui` and cohesive shared subsystems such as `data-table`, `editor`, and `layout` in
  their own directories.
- `src/hooks` and `src/lib` are feature-agnostic technical infrastructure only. A helper used by one
  capability belongs with that capability.
- Use the explicit `@components`, `@features`, `@hooks`, and `@lib` import roots. Root-level modules
  import one another relatively. Do not add `@/` imports or a catch-all `@*` alias that can capture
  scoped packages such as `@tanstack`.
- Mutable resources use thin `create.tsx` / `edit.tsx` shells plus `fields.tsx`. Do not use `mode`
  props for create/edit forms. Read-only resources use `list.tsx` and `detail.tsx`.
- Do not add barrels, feature re-exports, compatibility aliases, or alias-only local types.
- Use real generated/domain types directly. Do not add alias-only local types just to shorten names.

## Data / Forms

- API fetching uses generated operation functions and `unwrap` from `src/lib/api`.
- Query keys and invalidation use the owning feature's exported key factory. Keep one-off private keys
  beside their query instead of adding a registry.
- Lists use `components/data-table`.
- Every route-backed collection owns row-affecting table state in validated TanStack Router search
  params. This includes query text, faceted filters, sorting, page, and page size on list pages,
  detail-page result tables, and host subroutes.
- The leaf route that renders the collection owns its search schema. Build the common shape with
  `createTableSearchSchema`, strip canonical defaults with `stripSearchParams`, and bind the table
  through `useDataTableSearch`. Use functional `navigate({ search: updater, replace: true })`
  updates so table interactions preserve sibling search keys without filling browser history.
- `DataTableClient` processes a complete API response in the browser but still requires
  route-owned `tableState`. Do not add local query, filter, sort, or pagination fallbacks to shared
  table components. Keep only presentation state such as expansion, selection, and column
  visibility local.
- Local table/search state is reserved for transient workflows where a copied URL must not restore
  progress, such as form membership pickers and a running live-query session.
- Components must not imitate a missing or inconsistent API capability. A deliberately bounded,
  complete array response may be searched, sorted, and paginated client-side; do not treat an
  unpaginated array as permission to compensate for an unbounded collection in React. A
  server-paginated response must pass those controls to the generated API. If the generated
  contract cannot express the required filter, sort, or pagination behavior, fix the backend
  contract and regenerate it rather than synthesizing an approximation in React.
- Forms use `@tanstack/react-form`, zod schemas, `components/form-field.tsx`, and `components/form-actions.tsx`.
- Create/edit hooks toast success in `onSuccess`; mutation errors ride the global `MutationCache` toast.

## React

- Use `useEffect` only to sync with external systems: DOM APIs, subscriptions, timers, or network behavior.
- Avoid derived state in Effects. Calculate during render, or use `useMemo` for expensive computation.
- Put user-driven logic in event handlers. To reset state, prefer a `key` or render-time adjustment.
- Fetch Effects must guard stale responses with cleanup or abort behavior.

## Checks

- Frontend-only changes usually need `mise run //web:format`, `mise run //web:lint`, and `mise run //web:build`.
- API contract changes also need `mise run openapi-types`.
- Browser-visible UI changes should be checked in the in-app Browser when a dev server is available or starting one is in scope.
