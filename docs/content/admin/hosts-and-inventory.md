---
sidebar_position: 1
title: Hosts And Inventory
description: Host list, host detail, software inventory, labels, and directory-derived data.
---

# Hosts And Inventory

Hosts are the main admin object. The list endpoint supports pagination, search, status filters, label filters, software filters, ID filters, and osquery check-result filters.

## Admin Routes

| Route | Purpose |
| --- | --- |
| `GET /api/hosts` | List hosts. |
| `GET /api/hosts/{id}` | Load host detail. |
| `DELETE /api/hosts/{id}` | Delete one host. |
| `POST /api/hosts/bulk-delete` | Delete multiple hosts. |
| `GET /api/hosts/{id}/software` | List software observed on one host. |
| `PUT /api/hosts/{id}/user-affinity` | Set a manual user-affinity mapping. |
| `DELETE /api/hosts/{id}/user-affinity` | Remove manual user affinity. |

Host detail is assembled from the host base row and loaded children. Santa and Munki contribute host detail sections through handler-level contributors, so `hosts` does not import those packages.

## Labels

| Route | Purpose |
| --- | --- |
| `GET /api/labels` | List labels. |
| `POST /api/labels` | Create a label. |
| `GET /api/labels/{id}` | Load one label. |
| `PUT /api/labels/{id}` | Replace editable label state. |
| `DELETE /api/labels/{id}` | Delete a label. |

Manual labels carry selected host IDs. Dynamic labels carry an osquery query. Derived labels carry criteria such as directory department, group, or user.

## Software

| Route | Purpose |
| --- | --- |
| `GET /api/software` | List observed software titles. |
| `GET /api/software/{id}` | Load one software title with versions. |
| `GET /api/software/{id}/santa` | Load Santa reference data for the software title. |

Software is observed inventory. It is not Munki desired state.

## Directory Data

Directory routes expose synced users, groups, and departments:

- `GET /api/directory/users`
- `GET /api/directory/groups`
- `GET /api/directory/departments`

Directory sync starts only when the Entra tenant ID, client ID, and client secret are configured.
