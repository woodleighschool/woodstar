---
sidebar_position: 0
title: Overview
description: The admin API, how it's authenticated, and how this reference is generated.
---

# API Reference

This is the admin API: the JSON surface the web app runs on, served under `/api`. The pages beside this one are generated from Woodstar's OpenAPI spec and grouped by resource.

The Mac clients don't use this API. Their endpoints are a separate surface, documented in [Agent Protocols](../agent-protocols/overview).

## Authenticating

Two ways in, both tied to a Woodstar account:

- **Session cookie.** Signing in to the app sets `woodstar_session`, and the browser carries it. This is what the SPA uses.
- **API key.** For scripting, create an account API key and send it instead of using a session. Same permissions as the account it belongs to.

See [Authentication](../configuration/authentication) for both.

## Errors

Woodstar returns plain Huma error responses with a human-readable `message`. There's no sprawling error-code taxonomy to memorize; the message says what went wrong, and the frontend shows it.

## How this reference is generated

The spec is `web/openapi.yaml`, built by the backend itself:

```bash
go run ./cmd/woodstar openapi --output web/openapi.yaml
```

The same registration that mounts the routes produces the schema, so the reference can't drift from the server without the spec changing too. Regenerating these pages from the spec is covered in [Docs Site](../development/docs-site#the-api-reference).
