---
sidebar_position: 5
title: Munki Distribution Points
description: "Mirror nodes that hold package installers near a population of Macs and serve them on the LAN."
---

# Munki Distribution Points

Installer bytes are the heavy part of Munki. Manifests and catalogs are small, but a package payload can be hundreds of megabytes, and every Mac that's assigned it pulls the whole thing. Point a few hundred machines at one Woodstar, or at one S3 bucket, and that download traffic is the bottleneck, especially across a site link or a slow WAN.

A distribution point is a mirror you run near those machines. It holds copies of the installers and hands them out over the local network. Woodstar still decides what each Mac should have and still holds the canonical bytes; the distribution point is just a closer place to download from.

This is the one agent in Woodstar that isn't a Mac. It runs as a separate `woodstar mdp` process, authenticates with a per-distribution-point key instead of an agent secret, and keeps a long-lived connection open rather than polling. Because it isn't part of the admin API, these routes aren't in the [API reference](../api/overview); they're documented here.

## Who does what

| Actor                               | Responsibility                                                                                                                           |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Woodstar                            | Decides the desired package set, holds the canonical installers, signs download grants, picks which distribution point a Mac is sent to. |
| Distribution point (`woodstar mdp`) | Mirrors the installers, verifies them, reports its state, and serves them to redirected Macs.                                            |
| Munki client                        | Asks Woodstar for a package as usual, follows the redirect to a distribution point, downloads the bytes.                                 |

The distribution point never touches the database or storage credentials. It learns what to mirror over its connection and pulls the bytes through Woodstar.

## Running a worker

The worker is its own process and reads its own `WOODSTAR_MDP_` settings, with no database, session, or storage configuration. It needs the Woodstar URL, the key for the distribution point it's acting as, and a data directory to mirror into:

```bash
WOODSTAR_MDP_SERVER_URL=https://woodstar.example.com
WOODSTAR_MDP_SERVER_CA_FILE=/etc/woodstar/tls/woodstar-ca.pem
WOODSTAR_MDP_KEY=<key revealed on create or rotate>
WOODSTAR_MDP_DATA_DIR=/var/lib/woodstar-mdp
WOODSTAR_MDP_TLS_CERT_FILE=/etc/woodstar/tls/fullchain.pem
WOODSTAR_MDP_TLS_KEY_FILE=/etc/woodstar/tls/private-key.pem
woodstar mdp
```

`WOODSTAR_MDP_SERVER_CA_FILE` is only needed for a private Woodstar CA; it applies to the HTTPS and WebSocket clients. The worker serves Macs on `WOODSTAR_MDP_LISTEN_ADDR` (`:8080` by default). Set both worker TLS files for direct HTTPS, or leave both empty behind a TLS reverse proxy. The distribution point's client base URL must be its public HTTPS origin in either deployment. The full variable list is in [Environment](../configuration/environment#distribution-point-worker).

## The control channel

The worker holds one WebSocket open to Woodstar and reconnects with capped backoff if it drops.

| Method | Path                              | Auth                                                 |
| ------ | --------------------------------- | ---------------------------------------------------- |
| `GET`  | `/api/munki/distribution/connect` | `Authorization: Bearer <per-distribution-point key>` |

Three message types cross it:

- `hello`, sent once when the worker connects: the distribution point's identity and the full desired package set. Every connect is a clean reconciliation boundary, so a reconnect re-syncs everything.
- `desired_changed`, pushed whenever the desired set changes, for example after you edit a package or its installer finishes uploading.
- `state`, sent by the worker after it reconciles against a `hello` or `desired_changed`: one entry per package it holds, reporting `current` or, if a download failed, `error` with the reason.

The desired set is the whole available catalog: every package whose installer has finished uploading and has a known size and hash. Every distribution point mirrors all of it. Which Macs a given point actually serves is a separate decision, made at download time.

## Mirroring

When the worker receives a desired set, it reconciles its local copy. It downloads anything missing or changed, verifies each file's size and SHA-256 against what Woodstar reported, and deletes any installer no longer in the set. Downloads run a few at a time (`WOODSTAR_MDP_DOWNLOAD_CONCURRENCY`, default 4) and keep retrying with capped backoff after reporting an error.

It pulls the bytes through Woodstar rather than reaching into storage:

| Method | Path                                                         | Auth                                                 |
| ------ | ------------------------------------------------------------ | ---------------------------------------------------- |
| `GET`  | `/api/munki/distribution/packages/{package_id}/download-url` | `Authorization: Bearer <per-distribution-point key>` |

Woodstar returns a short-lived storage URL, so the worker downloads the file without ever holding storage credentials.

Verified state is snapshotted to a `0600` JSON file under the data directory, so a restart doesn't re-download everything. The snapshot is only an optimisation; the files on disk and the desired set from Woodstar are the real source of truth each cycle.

## Serving a Mac

When Woodstar redirects a Mac to a distribution point, it appends a grant:

```
GET {client_base_url}/munki/pkgs/{installer_item_location}?cap=<grant>
```

The grant is a storage capability signed with the distribution point's key. It carries the package id, the expected size and SHA-256, the installer path, the distribution point id, and an expiry fifteen minutes out. The worker verifies it offline with the same key, so a download needs no round trip back to Woodstar.

The size and hash in the grant bind it to specific bytes. The worker checks them against its mirror before serving, which catches the case where the package changed and the mirror hasn't caught up yet. It doesn't re-hash the file on every request: the bytes were hashed when they were downloaded, and Munki verifies `installer_item_hash` on the client anyway.

The serve path answers with a status that says which gate failed:

| Status        | Meaning                                                                 |
| ------------- | ----------------------------------------------------------------------- |
| `200` / `206` | Served, with range support for resumed downloads.                       |
| `401`         | Grant missing, invalid, or for a different package.                     |
| `410`         | Grant expired. The client re-requests the package and gets a fresh one. |
| `404`         | The package isn't mirrored here, or its file is missing.                |
| `409`         | The mirror is stale: its bytes don't match what the grant expects.      |
| `416`         | The requested byte range can't be satisfied.                            |

## How a Mac gets sent to a distribution point

The redirect happens inside the ordinary Munki package route. When a client requests a package, Woodstar looks for an eligible distribution point and, if it finds one, redirects there instead of serving the file itself. A point is eligible, in order, when:

1. It's enabled and has a client base URL set.
2. The client's IP falls inside one of its client CIDRs.
3. It reports the requested package as `current`, matching Woodstar's current installer for that package.
4. It's online, holding a live connection right now.

Distribution points are ordered, and the first eligible one wins, so the order is your preference list. If none qualifies, or Woodstar can't tell what the client's IP is, the download falls back to being served directly by Woodstar. Nothing breaks when a mirror is down, draining, or behind; the Mac just downloads from Woodstar instead. Working out the client's IP behind a proxy is what [`WOODSTAR_HTTP_CLIENT_IP_SOURCE`](../configuration/environment#client-ip) is for.

Icons are always served by Woodstar directly. Only package installers, the heavy downloads, go through distribution points.

The client side of this redirect is in [Munki Repository](./munki-repository#distribution-points); managing the distribution points themselves is in [Munki](../admin/munki#distribution-points).

## The key

Each distribution point has its own key. It's both the bearer credential the worker authenticates with and the secret Woodstar signs download grants with, so it's generated with the same entropy as an agent secret and stored in a form Woodstar can read back.

Woodstar shows the key once, when you create the point and when you rotate it, and never again. Put it straight into the worker's `WOODSTAR_MDP_KEY`. Rotating issues a new key and invalidates the old one; the worker picks it up on its next start.
