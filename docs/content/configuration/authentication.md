---
sidebar_position: 3
title: Authentication
description: Accounts, passwords, OIDC, API keys, and agent credentials.
---

# Authentication

Accounts can sign in with a local password or a configured OIDC provider. Agent credentials are managed separately.

## Create a local account

Run the user command with access to the database:

```bash
woodstar user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

The command prompts for a password unless `--password` is provided. The database URL comes from `WOODSTAR_DATABASE_URL` by default; use `--database-url` to pass another connection URL.

Email addresses must be lowercase. The built-in `admin` role grants full access; `viewer` grants read-only access to ordinary resources, excluding credentials, live queries and remediation operations.

Permissions combine direct and inherited role grants, taking the highest access for each resource: `none < view < edit`. A user without any grants cannot enter the app. The **Account** page shows the resulting permissions.

Use these commands to recover an existing local account:

```bash
woodstar user set-password --email you@example.com
woodstar user set-role --email you@example.com --role admin
```

## OIDC

OIDC is enabled when its issuer URL, client ID, and client secret are set. The configured email claim must match the email of a provisioned directory user. Authentication resolves that identity; application access still requires a direct or inherited permission grant.

See [Environment](./environment#oidc) for the settings.

## API keys

An account can create or rotate its API key from the **Account** page. The key has the same access as the account and can be used by scripts and [AutoPkg](../autopkg/overview).

## Agent credentials

Orbit, osquery, Santa, and Munki use [agent secrets](../concepts/agent-secrets), not account credentials.
