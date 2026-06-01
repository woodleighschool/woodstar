---
sidebar_position: 1
title: Admin API
description: Huma-backed admin API routes and OpenAPI generation.
---

# Admin API

The admin API is the session-authenticated JSON surface for the Woodstar SPA. It is registered with Huma under `/api` and documented as OpenAPI 3.1 by the `woodstar openapi` command.

Agent protocol routes are not part of this API contract. Orbit, osquery, Santa, and Munki protocol endpoints are documented under [Agent Protocols](../agent-protocols/overview).

## Generate OpenAPI

```bash
go run ./cmd/woodstar openapi --output web/openapi.yaml
```

The command builds the same Huma route registration used by the server. It does not call handlers and does not require a database.

To check the checked-in schema:

```bash
mise run test-openapi
```

To regenerate the OpenAPI file and frontend client types:

```bash
mise run openapi-types
```

## Route Groups

| Group | Routes |
| --- | --- |
| Auth | `/api/setup`, `/api/auth/session`, `/api/auth/login`, `/api/auth/logout`, SSO callback routes |
| Account | `/api/account`, `/api/account/api-key` |
| Users | `/api/users` |
| Hosts | `/api/hosts`, host software, user affinity, osquery host state, Santa host state |
| Labels | `/api/labels` |
| Software | `/api/software` |
| Directory | `/api/directory/users`, `/api/directory/groups`, `/api/directory/departments` |
| Agent secrets | `/api/agent-secrets` |
| osquery | `/api/osquery/reports`, `/api/osquery/checks`, `/api/live-queries` |
| Santa | `/api/santa/configurations`, `/api/santa/rules`, `/api/santa/events`, `/api/santa/file-access-events` |
| Munki | `/api/munki/software-titles`, package, deployment, and artifact subresources |

## Error Contract

Huma error messages are part of the SPA contract in this repository. The current API does not maintain a broad error-code taxonomy. Add narrow structured fields only when a real frontend flow needs to branch on them.

## Route Registration Rule

Route registration should remain side-effect-free. `api.BuildSchemaAPI` reuses the same route registration path with empty dependencies for schema generation.
