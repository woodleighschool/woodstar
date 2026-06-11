# Woodstar Docs

The Woodstar documentation site, built with [Docusaurus](https://docusaurus.io/). Pages live in `content/`. The API reference under `content/api/` is generated from `../web/openapi.yaml`, so don't edit those files by hand.

## Commands

```bash
pnpm install
pnpm start          # local dev server
pnpm build          # production build; also checks links and MDX
pnpm gen-api-docs   # regenerate the API reference from the OpenAPI spec
pnpm typecheck
```
