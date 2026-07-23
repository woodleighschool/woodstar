---
sidebar_position: 1
title: Run Woodstar
description: Run Woodstar and PostgreSQL with Docker Compose.
---

# Run Woodstar

The repository includes a Compose file for Woodstar and PostgreSQL. Woodstar uses the published `ghcr.io/woodleighschool/woodstar:rolling` image by default.

## Configuration

Start from the example environment file:

```bash
cp .env.example .env
```

Set these values in `.env`:

- `WOODSTAR_URL` to the HTTPS address used by browsers and Macs.
- `WOODSTAR_TLS_CERT_FILE` and `WOODSTAR_TLS_KEY_FILE` to certificate files on the Docker host.
- `WOODSTAR_STORAGE_CAPABILITY_KEY` to the output of `openssl rand -hex 32`.

The certificate must match `WOODSTAR_URL` and be trusted by the Macs connecting to Woodstar. Make sure the hostname resolves to the Docker host.

For development on one Mac, the [development setup](../development/setup) can generate a certificate and add the local hostname.

## Start Woodstar

```bash
docker compose up -d
```

Create the first account:

```bash
docker compose exec woodstar /woodstar user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

The command prompts for a password. When the account is ready, open the address set in `WOODSTAR_URL`.

## Start a distribution point worker

The Compose file includes the separate `woodstar mdp` worker under the `mdp` profile. After creating a distribution point, set `WOODSTAR_MDP_KEY` in `.env` and make sure its HTTPS URL matches the worker certificate, then start the profile:

```bash
docker compose --profile mdp up -d
```

The profile is disabled during an ordinary `docker compose up`. It stores cached installers in the `mdp-data` volume and publishes the worker on host port `8090`; set `WOODSTAR_MDP_PORT` to change that port.

## Run the current checkout

The `woodstar` service contains a commented `build` block and `pull_policy: build`. Uncomment both, then run:

```bash
docker compose up -d --build
```

This builds the same Dockerfile used for published images.

## Data

PostgreSQL data is stored in the `postgres-data` volume. Uploaded Munki files are stored in the `woodstar-data` volume, and distribution point cache data is stored in the `mdp-data` volume.

:::warning

`docker compose down --volumes` deletes the local database, uploaded files, and distribution point cache.

:::
