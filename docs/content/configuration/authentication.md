---
sidebar_position: 3
title: Authentication
description: Admin sign-in, first setup, OIDC, API keys, and how agent secrets differ.
---

# Authentication

There are two separate worlds of authentication in Woodstar, and it helps not to mix them up. People sign in to the admin app. Macs authenticate to the agent protocols. They don't share credentials.

## Admin sessions

Admin users are local Woodstar accounts in Postgres. Sign-in creates a server-side session, and the browser carries a cookie:

```text
woodstar_session
```

The cookie is secure by default. Set `WOODSTAR_SESSION_COOKIE_SECURE=false` only when the browser uses an HTTP development origin such as Vite on localhost.

The cookie value is an opaque random bearer token. Woodstar stores only its SHA-256 hash in Postgres; there is no session-signing key to configure or rotate. Revoking a session means deleting it or its database row, not rotating a storage credential.

## First setup

The very first account is created through the setup flow, which the app walks you through on a fresh install: start the backend, open it in a browser, and create the first local admin. After that, the normal login, logout, and session endpoints take over.

OIDC doesn't replace this first local account. You always have a local admin to fall back on.

## OIDC

OIDC is optional and switches on only when the issuer URL, client ID, and client secret are all configured. A partial block or failed provider discovery stops startup instead of silently disabling SSO.

The configured email claim becomes the user's identity in Woodstar. The settings are in [Environment](./environment#oidc).

## API keys

Each account can hold an API key for scripting against the admin API without a browser session. It's the same permissions as the account it belongs to. [AutoPkg](../autopkg/overview) uses an API key to push packages in.

## Agent secrets

Agent secrets are a different thing entirely. They aren't sessions and they aren't API keys: they're the shared credentials the Mac clients use to enroll and sync. Managing them is admin-only, and they get their own page in [Agent Secrets](../concepts/agent-secrets).
