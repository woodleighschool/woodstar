---
sidebar_position: 3
title: Authentication
description: Sign in with local accounts, an initial administrator, or OIDC.
---

# Authentication

Woodstar supports local accounts, an optional initial administrator, and OIDC. Administrators manage local accounts from the Directory page.

## Initial administrator

Set `WOODSTAR_INITIAL_ADMIN_EMAIL` and `WOODSTAR_INITIAL_ADMIN_PASSWORD` together to make an administrator available without creating an account first. It can manage users and reset passwords, but it does not appear in the Directory and has no Account page or API key.

If a regular account has the same email, the configured password takes precedence for password sign-in. Remove both settings and restart Woodstar to disable the initial administrator.

## OIDC

Configure an issuer URL, client ID, and client secret to enable OIDC. Only identities matching a Woodstar directory user can sign in. The initial administrator cannot sign in through OIDC. See [Environment](./environment#oidc).

## API keys

Regular accounts can create an API key with the same permissions as the account. [AutoPkg](../autopkg/overview) uses one to upload packages.

## Agent secrets

Mac clients use separate [Agent Secrets](../concepts/agent-secrets) to enroll and sync.
