---
sidebar_position: 5
title: Mutual TLS
description: Present client certificates for agent traffic at the network edge.
---

# Mutual TLS

The server can sit behind a reverse proxy or edge service that requires a trusted client certificate. The edge terminates mTLS; the server continues to authenticate Orbit and osquery with enrollment secrets and node keys, and Munki and Santa with bearer secrets.

The server does not receive or trust a forwarded client-certificate header. Configure the edge to require mTLS on the agent route families and forward accepted requests to it:

- `/api/fleet/orbit/`
- `/api/latest/fleet/device/`
- `/api/v1/osquery/`
- `/munki/`
- `/santa/sync/`

Keep the JSON API and browser routes separate unless every intended client can present the same identity.

## Orbit and osquery

`fleetctl package` can embed a PEM client certificate and private key while leaving the Fleet URL and enrollment secret in the system configuration profile:

```shell
fleetctl package \
  --type=pkg \
  --use-system-configuration \
  --fleet-tls-client-certificate=/path/to/client.crt \
  --fleet-tls-client-key=/path/to/client.key
```

The package installs the files as:

- `/opt/orbit/fleet_client.crt`
- `/opt/orbit/fleet_client.key`

Orbit uses that identity for its requests and configures its managed osquery process to use it too. The certificate requirement is additive: it does not replace the enrollment secret or issued node key.

For an independently managed osquery deployment, put the same PEM paths in its flagfile:

```text
--tls_client_cert=/opt/orbit/fleet_client.crt
--tls_client_key=/opt/orbit/fleet_client.key
```

If the fleetd package is not installed, deploy the certificate and unencrypted private key to another protected path and update both flags.

Fleet currently labels the two packaging flags as Fleet EE-licensed functionality. That label belongs to the upstream client packaging feature; it is not a server or edge capability check.

## Munki

Munki can use the same PEM files installed by the fleetd package. Add these keys to the `ManagedInstalls` payload:

```xml
<key>UseClientCertificate</key>
<true/>
<key>ClientCertificatePath</key>
<string>/opt/orbit/fleet_client.crt</string>
<key>ClientKeyPath</key>
<string>/opt/orbit/fleet_client.key</string>
```

Deploy the fleetd package before Munki first contacts the repository, and preserve the file ownership and permissions set by the package.

## Santa

Santa cannot consume the separate PEM certificate and key files used by Orbit, osquery, and Munki. It supports either:

- a PKCS#12 identity selected with `ClientAuthCertificateFile` and `ClientAuthCertificatePassword`; or
- an identity in the System Keychain selected with `ClientAuthCertificateCN` and, optionally, `ClientAuthCertificateIssuerCN`.

To reuse the same client identity, convert and deploy it as PKCS#12 or import the certificate and private key into the System Keychain. A generic keychain selection looks like:

```xml
<key>ClientAuthCertificateCN</key>
<string>CLIENT_CERTIFICATE_COMMON_NAME</string>
<key>ClientAuthCertificateIssuerCN</key>
<string>CLIENT_CERTIFICATE_ISSUER_COMMON_NAME</string>
```

The bearer secret remains in `SyncExtraHeaders`; mTLS is an additional edge check. If sharing the identity with Santa is not worth the extra packaging or keychain work, leaving Santa on HTTPS plus its bearer secret is a valid deployment choice.

## Upstream references

- [fleetd mTLS packaging and file locations](https://fleetdm.com/guides/fleetd-mtls)
- [fleetctl package flags](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/package.go)
- [osquery TLS client flags](https://osquery.readthedocs.io/en/stable/installation/cli-flags/#remote-settings-flags-optional)
- [Munki client certificate preferences](https://github.com/munki/munki/blob/main/code/client/munkilib/keychain.py)
- [Santa client authentication configuration](https://santa.dev/deployment/configuration.html)
