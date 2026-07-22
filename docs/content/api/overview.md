---
sidebar_position: 0
title: Overview
description: Authentication and generation for the API reference.
---

# API Reference

Woodstar's JSON API is served under `/api`. The web app uses the same endpoints documented here.

Mac clients use separate [agent protocols](../agent-protocols/overview).

## Authentication

The API accepts either the `woodstar_session` cookie created during sign-in or an account API key:

```http
Authorization: Bearer <api-key>
```

An API key has the same access as the associated account. Create or rotate a key from the **Account** page.

## Errors

Errors use `application/problem+json` and include a readable `detail`. Validation errors can also include field-level entries.

## Generation

The backend generates `web/openapi.yaml` from its registered routes. The docs site turns that schema into the operation pages in this section.

```bash
mise run openapi-types
mise run //docs:gen-api-docs
```

Generated API pages should not be edited by hand.
