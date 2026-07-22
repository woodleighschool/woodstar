---
sidebar_position: 5
title: Directory
description: Users and groups synced from Microsoft Entra ID.
---

# Directory

Woodstar can sync users and groups from Microsoft Entra ID. The sync runs when the tenant ID, client ID, and client secret are configured. See [Environment](../configuration/environment#entra-directory-sync).

Directory data is used for:

- **User affinity**, which associates a person with a Mac.
- **Derived labels**, which group hosts by user, group, or department.
- **OIDC sign-in**, after a synced user has been given a Woodstar role.

Set `WOODSTAR_ENTRA_TRANSITIVE_GROUPS=true` if nested group membership should be expanded.

Local Woodstar accounts and Entra users appear in the same directory. Entra sync does not copy passwords or create browser sessions.
