---
sidebar_position: 3
title: Authentication
description: Provision local users and sign in with passwords, OIDC, or API keys.
---

# Authentication

Every person who signs in to Woodstar is a persisted directory user. Administrators manage users from the Directory page; operators can create or recover them with the Woodstar binary when no administrator can sign in.

## Provision a local user

Run the user command against the Woodstar database before exposing a fresh server:

```bash
woodstar user create \
  --email admin@example.com \
  --name Administrator \
  --role admin
```

The command prompts for the password when `--password` is omitted. Pass `--password` explicitly for non-interactive automation. It reads `WOODSTAR_DATABASE_URL`, or accepts `--database-url`. Email addresses must be lowercase; surrounding whitespace is ignored.

The release image is distroless but still runs Woodstar subcommands directly. It does not need a shell:

```bash
kubectl exec -it deploy/woodstar -- \
  /woodstar user create \
  --email admin@example.com \
  --name Administrator \
  --role admin
```

Use `user set-password --email ...` to replace a local password and `user set-role --email ... --role admin` to restore administrator access. Woodstar permits deleting or demoting the final administrator; the same commands recover that deliberate zero-administrator state.

## OIDC

Configure an issuer URL, client ID, and client secret to enable OIDC. Only identities whose configured claim exactly matches a Woodstar user's email can sign in, and that user must have an app role. A directory UPN remains metadata rather than a second login identifier. See [Environment](./environment#oidc).

## API keys

Users can create an API key with the same permissions as their account. [AutoPkg](../autopkg/overview) uses one to upload packages.

## Agent secrets

Mac clients use separate [Agent Secrets](../concepts/agent-secrets) to enroll and sync.
