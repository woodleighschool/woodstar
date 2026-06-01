---
sidebar_position: 3
title: Authentication
description: Local users, sessions, OIDC gating, API keys, and agent secrets.
---

# Authentication

Woodstar has two separate authentication areas: admin users and agent protocols.

## Admin Sessions

Admin users are local Woodstar accounts stored in Postgres. The server uses `scs` with the pgx session store. The browser cookie is named:

```text
woodstar_session
```

The cookie is marked secure when `WOODSTAR_PUBLIC_URL` uses `https`.

## First Setup

The first account is created through `/api/setup`. After setup, normal session routes handle login, logout, and session inspection:

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/session`

Account routes expose the signed-in user's account state and API key management:

- `GET /api/account`
- `PUT /api/account`
- `POST /api/account/api-key`
- `DELETE /api/account/api-key`

## OIDC

OIDC is capability-gated. It is enabled only when issuer URL, client ID, and client secret are all configured. The service discovers the provider at startup. If discovery fails, SSO stays disabled and local auth continues.

The configured email claim becomes the local user identity used by Woodstar.

## Agent Secrets

Agent secrets are not browser sessions and not API keys. They are shared credentials accepted by Orbit/osquery enrollment, Santa sync, and Munki repository access.

Admin routes for agent secrets are protected by admin middleware.
