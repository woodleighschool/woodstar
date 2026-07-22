# Woodstar documentation

The documentation site is built with [Docusaurus](https://docusaurus.io/). Hand-written pages live in `content/`; the API reference in `content/api/` is generated from `../web/openapi.yaml`.

## Writing rules

- Describe current Woodstar behaviour. Check the code, UI, or generated API before writing.
- Write for someone using or working on Woodstar, not for invented roles or customer types.
- Start with the task or concept the page is for. Keep background to what helps with that task.
- Prefer plain statements and exact names. Avoid marketing copy, maturity disclaimers, design commentary, and answers to one-off development questions.
- Put commands, settings, and protocol details in their reference page. Link to them instead of repeating them elsewhere.
- Keep comparisons to other projects factual and local, such as protocol compatibility or upstream attribution.
- Do not edit generated files under `content/api/`.

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
