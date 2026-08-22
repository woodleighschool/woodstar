---
sidebar_position: 4
title: Munki Repository
description: Configure Munki to use the server as its repository.
---

# Munki Repository

The server serves manifests, catalogs, packages, icons, and Managed Software Center resources using Munki's repository layout.

## Configure Munki

Create a secret from the **Munki** overview, then replace the example URL and secret in this profile:

```xml title="munki.mobileconfig"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>AdditionalHttpHeaders</key>
      <array>
        <string>Authorization: Bearer REPLACE_WITH_SECRET</string>
        <string>X-Woodstar-Serial-Number: $SERIALNUMBER</string>
      </array>
      <key>FollowHTTPRedirects</key>
      <string>https</string>
      <key>SoftwareRepoURL</key>
      <string>https://woodstar.example.com/munki</string>
      <key>PayloadDisplayName</key>
      <string>Munki</string>
      <key>PayloadIdentifier</key>
      <string>com.example.woodstar.munki.managedinstalls</string>
      <key>PayloadType</key>
      <string>ManagedInstalls</string>
      <key>PayloadUUID</key>
      <string>EF6B0B39-B2BE-44F7-A2B5-5F49282B221D</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDescription</key>
  <string>Configures Munki for managed software.</string>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Munki</string>
  <key>PayloadIdentifier</key>
  <string>com.example.woodstar.munki</string>
  <key>PayloadOrganization</key>
  <string>Example Organization</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>56E74DA2-6F02-4E85-8C95-BA51C34F88F0</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

The profile sets:

- `SoftwareRepoURL` to `<WOODSTAR_URL>/munki`
- `FollowHTTPRedirects` to `https`
- `Authorization: Bearer <MUNKI_SECRET>` in `AdditionalHttpHeaders`
- `X-Woodstar-Serial-Number: <MDM_EXPANDED_SERIAL>` in `AdditionalHttpHeaders`

Both headers accompany every repository request. The bearer secret authenticates the Munki
client and the serial header binds the request to an enrolled host. Munki does not
need a `ClientIdentifier`, UUID field, or preflight/postflight hook.

See [Mutual TLS](./mutual-tls#munki) to have Munki present the same PEM identity installed with fleetd.

## Routes

| Method | Path                              | Purpose                           |
| ------ | --------------------------------- | --------------------------------- |
| `GET`  | `/munki/manifests/{name}`         | Host-scoped manifest              |
| `GET`  | `/munki/catalogs/woodstar`        | Host-scoped package catalog       |
| `GET`  | `/munki/icons/_icon_hashes.plist` | Host-scoped icon hash list        |
| `GET`  | `/munki/pkgs/*`                   | Package installer                 |
| `GET`  | `/munki/icons/*`                  | Software icon                     |
| `GET`  | `/munki/client_resources/*`       | Managed Software Center resources |

Path components select a Munki repository object; they do not identify the requesting Mac.
For example, `manifests/{name}` is a manifest selector and a package path selects an installer.
The server always resolves the requester from `X-Woodstar-Serial-Number` before serving the
selected object.

A missing or invalid bearer secret returns `401`. A missing or unknown serial, an unavailable
repository object, or an object not assigned to that host returns `404`; this deliberately
blackholes objects outside the host's repository view.

## Manifests and catalogs

The server builds each manifest and `woodstar` catalog from the host's matching software targets.
A latest-version target uses the bare Munki name; a pinned package uses `name--version`.

Installer-backed catalog items include their location, size, and SHA-256. `nopkg` items omit
the installer fields. Package and icon files are available only when the selected item belongs
to the requesting host's catalog.

## Files

With file storage, the server streams artifacts. With S3, it redirects to a presigned URL. Matching package requests may be redirected to a [distribution-point cache](./munki-distribution); icons and client resources always use primary storage.

Client Resources is one shared archive, but the server still resolves and authorizes the host
before it serves the archive. Munki may request any Client Resources path; the path is only a
repository selector. When no archive is deployed, the route returns `404` and Munki uses its
built-in resources.
