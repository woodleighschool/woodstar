# Documentation

The documentation site is built with [Docusaurus](https://docusaurus.io/).

Pages live in `content/`; the API reference in `content/api/` is generated from `../web/openapi.yaml`.

## Commands

Run these from `docs/`:

```bash
pnpm install
pnpm start
pnpm typecheck
pnpm build
pnpm gen-api-docs
```

`pnpm build` checks the production site, links, and MDX. Regenerate the API pages after `web/openapi.yaml` changes.
