---
sidebar_position: 4
title: Munki
description: Woodstar-managed Munki titles, packages, deployments, artifacts, and client repository output.
---

# Munki

Munki in Woodstar is desired state. It is separate from observed software inventory.

The current admin model has four main objects:

| Object | Purpose |
| --- | --- |
| Software title | Human-owned grouping for one managed software item. |
| Package | One pkginfo item or imported pkginfo payload. |
| Deployment | Assignment behavior and scope for a software title. |
| Artifact | Package or icon object metadata used by rendered Munki pkginfo. |

## Software Titles

| Route | Purpose |
| --- | --- |
| `GET /api/munki/software-titles` | List titles. |
| `POST /api/munki/software-titles` | Create a title. |
| `GET /api/munki/software-titles/{id}` | Load a title with packages and deployments. |
| `PATCH /api/munki/software-titles/{id}` | Update title metadata. |

## Packages

| Route | Purpose |
| --- | --- |
| `GET /api/munki/software-titles/{id}/packages` | List packages for a title. |
| `POST /api/munki/software-titles/{id}/packages` | Create package metadata. |
| `POST /api/munki/software-titles/{id}/packages/import` | Import an existing pkginfo item. |
| `GET /api/munki/software-titles/{id}/packages/{package_id}` | Load one package. |
| `PATCH /api/munki/software-titles/{id}/packages/{package_id}` | Update one package. |

Package fields intentionally preserve a lot of Munki vocabulary: installer type, restart action, blocking applications, requires, update-for, supported architectures, unattended flags, optional `extra_pkginfo`, and artifact references.

## Deployments

| Route | Purpose |
| --- | --- |
| `GET /api/munki/software-titles/{id}/deployments` | List deployments for a title. |
| `POST /api/munki/software-titles/{id}/deployments` | Create a deployment. |
| `GET /api/munki/software-titles/{id}/deployments/{deployment_id}` | Load one deployment. |
| `PATCH /api/munki/software-titles/{id}/deployments/{deployment_id}` | Update one deployment. |
| `PUT /api/munki/software-titles/{id}/deployments/order` | Reorder deployments. |

Deployments can target all hosts, include/exclude labels, and include/exclude concrete host IDs. Actions are `install`, `remove`, `update_if_present`, and `none`.

## Artifacts

| Route | Purpose |
| --- | --- |
| `POST /api/munki/artifacts` | Register an existing package or icon artifact. |
| `POST /api/munki/artifacts/upload-url` | Create a temporary object-storage upload URL. |
| `GET /api/munki/artifacts/{id}/content` | Fetch or redirect artifact content. |

Artifact upload URLs require configured Munki S3 storage. Without that backend, metadata can exist but upload and redirect flows are limited.
