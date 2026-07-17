---
sidebar_position: 2
title: Docker Compose
description: Run the checked-in PostgreSQL service for local Woodstar development.
---

# Docker Compose

The root `docker-compose.yml` provides PostgreSQL 18 for local development and tests. It is not a Woodstar deployment or production chart; run Woodstar through the [mise development tasks](./local-development).

## Services

| Service    | Purpose           | Published Port |
| ---------- | ----------------- | -------------- |
| `postgres` | Woodstar database | `5432:5432`    |

## Start PostgreSQL

```bash
docker compose up -d postgres
```

The default local database URL is:

```text
postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable
```

Woodstar defaults to file storage during development. The S3 integration test starts its own ephemeral Garage container, so Compose does not carry an object-storage service or development credentials.

## Volumes

Compose keeps the database in one named volume:

```text
postgres-data
```

`docker compose down` preserves it. Use `docker compose down --volumes` only when you intentionally want to discard the local database.
