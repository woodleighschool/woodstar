---
sidebar_position: 5
title: Munki Distribution Points
description: Run a local cache for Munki package installers.
---

# Munki Distribution Points

A distribution point is a local cache for Munki package installers. Woodstar redirects matching package requests to the cache's configured HTTPS URL. The Munki client then downloads the installer from the cache.

## Create a distribution point

Under **Munki > Distribution Points**:

1. Create a point with an HTTPS URL that its Munki clients can resolve and reach.
2. Add the client CIDRs that should use the cache.
3. Copy the key shown after creation.
4. Start `woodstar mdp` on the machine that will hold the cached installers.

Woodstar evaluates points in their displayed order. A package is redirected only when the point is enabled, the worker is connected, the client's source IP matches one of the configured CIDRs, and the cached package is current.

:::warning

Woodstar cannot test whether a Munki client can reach the cache URL. Woodstar knows that the worker is connected and has the package, but cannot see DNS, routing, firewall, or TLS failures between the client and the cache. If that connection fails after the redirect, the package download fails; Woodstar cannot send the same request back through primary storage.

Only assign client CIDRs that can resolve and connect to the configured URL over HTTPS.

:::

## Run the worker

```bash
WOODSTAR_MDP_SERVER_URL=https://woodstar.example.com
WOODSTAR_MDP_KEY=<distribution-point-key>
WOODSTAR_MDP_DATA_DIR=/var/lib/woodstar-mdp
WOODSTAR_MDP_TLS_CERT_FILE=/etc/woodstar/tls/fullchain.pem
WOODSTAR_MDP_TLS_KEY_FILE=/etc/woodstar/tls/private-key.pem
woodstar mdp
```

The worker listens on `:8080` by default. Set both worker TLS files for direct HTTPS, or leave both empty behind a reverse proxy.

See [Environment](../configuration/environment#distribution-point-worker) for every worker setting.

## Caching

The worker keeps a WebSocket connection to Woodstar and receives the complete desired package list. Missing or changed installers are cached and checked by size and SHA-256. Installers no longer wanted are removed.

Cache state is kept in the data directory, and downloads resume after a restart. The Woodstar UI reports each package as pending, syncing, current, or in error.

The worker uses the distribution-point key to connect and request short-lived download URLs. Database and storage credentials stay on the server.

## Serving packages

Munki requests packages from Woodstar as usual. When a matching point has the package, Woodstar redirects the request with a signed, short-lived grant. The worker validates the grant and supports byte ranges for resumed downloads.

If the worker is offline, the cached package is stale, or the client's address does not match the configured CIDRs, Woodstar serves the package from primary storage instead. This fallback happens before a redirect and cannot help when the selected cache is unreachable from the client.

Rotating a distribution-point key disconnects the old worker and invalidates its download grants. Update `WOODSTAR_MDP_KEY` before restarting the worker.
