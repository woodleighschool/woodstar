# AGENTS.md

Frontend rules for work under `web/`.

## Stack / Commands

- React 19, Vite, TypeScript, Tailwind v4, TanStack Router/Form/Query/Table, shadcn-style primitives, lucide icons.
- Source: `web/src`; static assets: `web/public`; production bundle: `web/dist`.
- From repo root: `mise //web:dev`, `mise //web:build`, `mise //web:lint`, `mise //web:format`, `mise run openapi-types`, `mise run test-openapi`.
- Inside `web/`: `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm format`, `pnpm openapi:types`.
- `web/openapi.yaml` and `web/src/lib/api-client/` are generated. Do not edit generated client files by hand.

## Product / UI

- This is a self-hosted admin SPA served by the Go backend. No marketing pages, onboarding filler, SaaS ceremony, or speculative multi-tenant UI.
- Use established app chrome: `PageShell`, `PageHeader`, sidebar, breadcrumbs, and existing layout components.
- Use CSS variable tokens and existing UI primitives. Avoid hardcoded hex colors or raw palette utilities.
- Use lucide icons for common actions. Keep operational copy short.
- Match the existing dense admin feel. Do not nest cards inside cards or turn page sections into floating cards.

## Organization

- Resource pages live in `src/pages/<resource>/`.
- Mutable resources use thin `create.tsx` / `edit.tsx` shells plus `fields.tsx`. Do not use `mode` props for create/edit forms.
- Read-only resources use `list.tsx` and `detail.tsx`.
- Routes live in `src/routes`.
- Hooks live flat in `src/hooks/use-<resource>.ts`.
- Shared technical utilities and cross-capability domain helpers live in `src/lib/`; feature-private schemas/components stay with the feature.
- Use real generated/domain types directly. Do not add alias-only local types just to shorten names.

## Data / Forms

- API fetching uses generated operation functions and `unwrap` from `src/lib/api`.
- Query keys and invalidation use `src/lib/query-keys`; do not inline query key arrays.
- Lists use `components/data-table`.
- Forms use `@tanstack/react-form`, zod schemas, `components/form-field.tsx`, and `components/form-actions.tsx`.
- Create/edit hooks toast success in `onSuccess`; mutation errors ride the global `MutationCache` toast.

## React

- Use `useEffect` only to sync with external systems: DOM APIs, subscriptions, timers, or network behavior.
- Avoid derived state in Effects. Calculate during render, or use `useMemo` for expensive computation.
- Put user-driven logic in event handlers. To reset state, prefer a `key` or render-time adjustment.
- Fetch Effects must guard stale responses with cleanup or abort behavior.

## Checks

- Frontend-only changes usually need `mise //web:format`, `mise //web:lint`, and `mise //web:build`.
- API contract changes also need `mise run openapi-types` and `mise run test-openapi`.
- Browser-visible UI changes should be checked in the in-app Browser when a dev server is available or starting one is in scope.
