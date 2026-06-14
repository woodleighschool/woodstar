# AGENTS.md

Frontend rules for AI agents working under `web/`.

## Stack And Commands

- React 19, Vite, TypeScript, Tailwind v4, TanStack Router/Form/Query/Table, shadcn-style components, lucide icons.
- Source lives in `web/src`; static assets live in `web/public`; production bundle output is `web/dist`.
- From repo root, prefer mise tasks: `mise run frontend`, `mise run frontend-lint`, `mise run frontend-format`, `mise run openapi-types`, `mise run test-openapi`.
- Inside `web/`, use `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm format`, and `pnpm openapi:types`.
- `web/openapi.yaml` and `web/src/lib/api-client/` are generated contract files. Do not edit generated client files by hand.

## Product Shape

- This is a self-hosted admin SPA served by the Go backend. No marketing pages, onboarding filler, SaaS ceremony, or speculative multi-tenant UI.
- Prefer terse, operational copy. If the control or nearby heading already explains the object, button labels can be just `Create`, `Save`, or `Delete`.
- Every page should sit inside the established app chrome: `PageShell`, `PageHeader`, sidebar, breadcrumbs, and existing layout components.
- Use CSS variable tokens and existing UI primitives. Avoid hardcoded hex colors or raw palette utility classes.
- Use lucide icons for common actions. Icon-only buttons are for common or cramped repeated actions; otherwise use text or icon+text.

## Organization

- Resource pages live in `src/pages/<resource>/`.
- Mutable resources use thin `create.tsx` and `edit.tsx` shells plus `fields.tsx` for schema, zod validation, mappers, and shared form UI. Do not use `mode` props for create/edit forms.
- Read-only resources use `list.tsx` and `detail.tsx`.
- Routes live in `src/routes`. Paginated index routes spread `tableSearchSchema.shape` into `validateSearch`.
- Hooks live flat in `src/hooks/use-<resource>.ts`. Do not create per-capability hook folders or pack unrelated resources into one hook file.
- All API fetching goes through `apiClient` and `unwrap` from `src/lib/api`.
- Query keys and invalidation use `src/lib/query-keys`; do not inline query key arrays.
- `src/lib/` is for resource-agnostic technical utilities and cross-capability domain surfaces. Capability-private schemas, metadata, and sub-components stay in the owning feature folder.
- Use real generated/domain types directly. Do not add alias-only local types just to shorten names.

## Data, Forms, And Feedback

- Lists use `components/data-table` and its search, filter, column, bulk, skeleton, and empty-state helpers.
- A first-class resource list renders its empty state as chrome (icon + title + description). Subresources, tabs, nested tables, and detail-page sections use plain text instead.
- Query load failures render `QueryError` with retry. Detail/form first-load can return `null` instead of flashing a skeleton.
- Forms use `@tanstack/react-form` with `validationLogic: revalidateLogic()` and one form-level `validators: { onDynamic: schema }` (a zod schema, or a function returning `{ fields }` when the schema only covers a subset of the values). Validation runs on submit, then live on change. Build fields with `components/form-field.tsx` and the submit/cancel footer with `components/form-actions.tsx`.
- `FormActions` gates the submit button on the form's own state (`canSubmit`, `isDefaultValue`); there are no save spinners. Forms that keep part of their state outside the form (uploads, separate editors) pass `requireDirty={false}`.
- Mutations report their own outcome. Create/edit hooks `toast.success` in `onSuccess`; errors ride the global `MutationCache` toast. Pure actions such as delete, copy, rotate, reorder, and bulk operations `toast.success` at the call site.
- Field validation shows inline under each field; submit and server errors toast. Only the pre-auth login/setup forms keep an inline error, via mutation `meta: { inlineError: true }`.

## React Effects

- Use `useEffect` only to sync with external systems: DOM APIs, subscriptions, timers, or network behavior.
- Avoid derived state in Effects. Calculate during render, or use `useMemo` for expensive computation.
- Put user-driven logic in event handlers.
- To reset state, prefer a `key` or render-time adjustment.
- Fetch Effects must guard against stale responses with cleanup or abort behavior.

## UI Checks

- Match the existing density and admin feel before inventing new layout language.
- Keep text inside buttons, tabs, cards, tables, and sidebars from wrapping badly or overlapping at mobile and desktop widths.
- Do not nest cards inside cards or turn page sections into floating cards.
- Browser-visible UI changes should be checked in the in-app Browser when a dev server is available or starting one is in scope. If not checked, say so.

## Tests And Final Report

- Add frontend tests only when existing tooling covers the behavior and the behavior is worth protecting. Do not introduce a new test stack for one trivial component.
- Before handoff, run the narrowest useful checks, usually `mise run frontend-format`, `mise run frontend-lint`, and `mise run frontend` for frontend-only work.
- For API contract changes, also run `mise run openapi-types` and `mise run test-openapi`.
- Final responses should state checks run, checks skipped with a reason, and any unresolved failure.
