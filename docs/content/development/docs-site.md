---
sidebar_position: 3
title: Docs Site
description: Work on the Woodstar documentation site.
---

# Docs Site

The documentation site is built with [Docusaurus](https://docusaurus.io/) from files under `docs/`.

## Layout

```text
docs/
  content/              hand-written pages
  content/api/          generated API reference
  src/pages/index.tsx   home page
  src/css/custom.css    site styles
  static/img/           images
  docusaurus.config.ts  site configuration
  sidebars.ts           sidebar configuration
```

## Commands

Run from the repository root:

```bash
mise run //docs:dev
mise run //docs:lint
mise run //docs:format
mise run //docs:build
```

The build regenerates the API pages and checks links and MDX.

## API reference

The API operation pages are generated from `web/openapi.yaml` with `docusaurus-openapi-docs`.

```bash
mise run //docs:gen-api-docs
```

Do not edit generated files under `docs/content/api/`. Change the backend contract, run `mise run openapi-types`, then regenerate the docs.

## Writing

Follow the [documentation writing rules](https://github.com/woodleighschool/woodstar/blob/main/docs/README.md#writing-rules). Keep task instructions near the task, use the names shown in the app, and link to reference pages instead of copying their tables.
