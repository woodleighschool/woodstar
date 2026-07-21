---
sidebar_position: 5
title: Directory
description: Synced users, groups, and departments from Entra, and what they feed.
---

# Directory

Directory brings people into Woodstar. When Entra sync is configured, Woodstar pulls users, groups, and departments on a schedule, and you can browse them under Directory in the admin app.

This is optional. Sync only starts when the Entra tenant ID, client ID, and client secret are all set; leave them unset and the directory views stay empty without getting in the way. See [Environment](../configuration/environment#entra-directory-sync) for the settings.

## What it feeds

Directory data isn't just a list to look at. Two things lean on it:

- **Derived labels** can match on directory attributes, so "everyone in the Science department" becomes a label you target Munki and Santa at.
- **User affinity** gets richer. Once Woodstar has resolved which person a Mac belongs to, directory data fills in their name, department, and groups.

Group expansion can be made transitive with `WOODSTAR_ENTRA_TRANSITIVE_GROUPS` if you need nested group membership to count.

This is read-only enrichment. Directory sync never logs anyone in, and a synced user is not a Woodstar account. Admin sign-in is covered in [Authentication](../configuration/authentication).
