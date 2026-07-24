---
sidebar_position: 5
title: Munki Distribution Points
description: Run a local cache for Munki package installers.
---

# Munki Distribution Points

A distribution point is a local cache for Munki package installers. Woodstar redirects matching package requests to the cache's configured HTTPS URL. The Munki client then downloads the installer from the cache.

## How client matching works

Woodstar selects a distribution point using the client IP derived from the current package request. It evaluates the address on every request rather than using an interface address reported by osquery. Request-time matching follows a client as it changes networks and avoids routing from stale inventory.

The address depends on how the client reaches Woodstar:

| Request path                                    | Address the client CIDR must match                                                                                     |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Directly over the same private network          | The client's private source address, such as `10.20.0.0/16`                                                            |
| Directly over the internet                      | The public egress address after NAT, such as `203.0.113.42/32`                                                         |
| Through a correctly configured reverse proxy    | The client address selected from a trusted forwarded header, commonly the client's public egress address               |
| Through a proxy without client-IP configuration | The proxy's connection address; unrelated clients can appear to come from the same source and match the wrong location |

If clients can reach Woodstar over both IPv4 and IPv6, assign the relevant ranges for both protocols.

The default `remote_addr` source reads the peer connected directly to Woodstar. Before assigning client CIDRs behind a proxy, configure [Client IP](../configuration/environment#client-ip) so Woodstar trusts only the proxy's sanitized forwarded address. Otherwise, the CIDRs describe the proxy network rather than the Munki clients.

### Example: internet-hosted Woodstar with an on-premises cache

Suppose a campus has the public egress address `203.0.113.42`, Woodstar sits behind Cloudflare, and `mdp.example.com` is reachable only from the campus:

1. Restrict the Woodstar origin to the Cloudflare path. Cloudflare supplies the connecting client address in `CF-Connecting-IP`.
2. Configure Woodstar with `WOODSTAR_HTTP_CLIENT_IP_SOURCE=header` and `WOODSTAR_HTTP_CLIENT_IP_HEADER=CF-Connecting-IP`.
3. Assign `203.0.113.42/32` to the distribution point.
4. Ensure campus clients can resolve and reach `https://mdp.example.com`.

Campus requests match the cache using their current public egress address. Off-campus requests do not match and Woodstar serves those packages from primary storage. If clients instead reach Woodstar directly over the campus network, keep the default client-IP source and assign the relevant private subnet.

## Create a distribution point

Under **Munki > Distribution Points**:

1. Create a point with an HTTPS URL that its Munki clients can resolve and reach.
2. Add the client CIDRs that should use the cache.
3. Copy the key shown after creation.
4. Start `woodstar mdp` on the machine that will hold the cached installers.

Woodstar evaluates points in their displayed order. A package is redirected only when the point is enabled, the worker is connected, the derived client IP matches one of the configured CIDRs, and the cached package is current.

:::warning

Woodstar cannot test whether a Munki client can reach the cache URL. Woodstar knows that the worker is connected and has the package, but cannot see DNS, routing, firewall, or TLS failures between the client and the cache. If that connection fails after the redirect, the package download fails; Woodstar cannot send the same request back through primary storage.

Only assign client CIDRs that can resolve and connect to the configured URL over HTTPS.

:::

## Run the worker

Download the archive for the worker host from [GitHub Releases](https://github.com/woodleighschool/woodstar/releases), extract it, and place `woodstar` on `PATH`. Releases include Linux and macOS archives for AMD64 and ARM64.

```bash
WOODSTAR_MDP_SERVER_URL=https://woodstar.example.com
WOODSTAR_MDP_KEY=<distribution-point-key>
WOODSTAR_MDP_DATA_DIR=/var/lib/woodstar-mdp
WOODSTAR_MDP_TLS_CERT_FILE=/etc/woodstar/tls/fullchain.pem
WOODSTAR_MDP_TLS_KEY_FILE=/etc/woodstar/tls/private-key.pem
woodstar mdp
```

The worker listens on `:8080` by default. Set both worker TLS files for direct HTTPS, or leave both empty behind a reverse proxy.

The repository Compose file can run the same worker from the published image. Set the worker values in `.env`, then enable its profile:

```bash
docker compose --profile mdp up -d
```

The `mdp` profile is disabled during an ordinary `docker compose up`.

See [Environment](../configuration/environment#distribution-point-worker) for every worker setting.

## Caching

The worker keeps a WebSocket connection to Woodstar and receives the complete desired package list. Missing or changed installers are cached and checked by size and SHA-256. Installers no longer wanted are removed.

The WebSocket upgrade requires an exact protocol match. Woodstar rejects mismatched workers instead of attempting compatibility, and the UI marks them as incompatible. The server and worker also exchange their Woodstar build versions for diagnostics; build versions do not control protocol compatibility. Run the same Woodstar release for both processes.

Cache state is kept in the data directory, and downloads resume after a restart. The Woodstar UI reports each package as pending, syncing, current, or in error.

The worker uses the distribution-point key to connect and request short-lived download URLs. Database and storage credentials stay on the server.

## Serving packages

Munki requests packages from Woodstar as usual. When a matching point has the package, Woodstar redirects the request with a signed, short-lived grant. The worker validates the grant and supports byte ranges for resumed downloads.

If the worker is offline, the cached package is stale, or the client's address does not match the configured CIDRs, Woodstar serves the package from primary storage instead. This fallback happens before a redirect and cannot help when the selected cache is unreachable from the client.

Rotating a distribution-point key disconnects the old worker and invalidates its download grants. Update `WOODSTAR_MDP_KEY` before restarting the worker.
