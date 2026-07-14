---
sidebar_position: 2
title: Docker Compose
description: Run the checked-in compose stack for local Woodstar, Postgres, and Garage.
---

# Docker Compose

The root `docker-compose.yml` is a local stack, not a production chart. It builds the Woodstar image from the repository, starts Postgres 18, and starts Garage as an S3-compatible backend for Munki artifacts.

## Services

| Service    | Purpose                                  | Published Ports                |
| ---------- | ---------------------------------------- | ------------------------------ |
| `woodstar` | Go server plus built frontend over HTTPS | `8080:8080`                    |
| `postgres` | Woodstar database                        | `5432:5432`                    |
| `garage`   | Local object storage for Munki artifacts | `3900`, `3901`, `3902`, `3903` |

## Start The Stack

```bash
cp .env.example .env
mise run dev-tls
docker compose up --build
```

The compose file overrides the host-development ports and URLs so Woodstar serves the built SPA and API together at `https://localhost:8080`. It also points the app at Postgres by service name, mounts only the generated leaf and key, and switches the example file storage to Garage.

Run `mise run dev-tls-trust` first if the Compose URL needs to be trusted by local browsers.

This direct TLS setup is for local Compose. A production container normally leaves `WOODSTAR_TLS_CERT_FILE` and `WOODSTAR_TLS_KEY_FILE` empty and receives private HTTP traffic from its HTTPS reverse proxy.

## Munki Object Storage Defaults

The compose stack runs Garage as the S3 backend, so it sets `WOODSTAR_STORAGE_KIND=s3` and points at Garage with development credentials:

```bash
WOODSTAR_STORAGE_KIND=s3
WOODSTAR_STORAGE_S3_BUCKET=woodstar-munki
WOODSTAR_STORAGE_S3_REGION=garage
WOODSTAR_STORAGE_S3_ENDPOINT=http://garage:3900
WOODSTAR_STORAGE_S3_PUBLIC_ENDPOINT=http://127.0.0.1:3900
WOODSTAR_STORAGE_S3_PATH_STYLE=true
```

Do not carry those credentials into a real deployment. They exist so artifact upload and redirect paths can be exercised locally without another storage service. Drop `WOODSTAR_STORAGE_KIND` (or set it to `file`) and Woodstar falls back to the on-disk backend, which needs no bucket at all.

## Volumes

The compose stack creates named volumes for Postgres and Garage:

```text
postgres-data
garage-data
```

Remove them only when you intentionally want to throw away local database and artifact state.
