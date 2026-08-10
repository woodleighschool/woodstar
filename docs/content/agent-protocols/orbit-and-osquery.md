---
sidebar_position: 2
title: Orbit and osquery
description: Enroll Macs and run osquery through Fleet-compatible endpoints.
---

# Orbit and osquery

Orbit and osquery enroll with the Orbit agent secret, then use a per-host node key. Woodstar supports the Fleet route and response shapes used by both clients.

## Configure Orbit

Create a host enrollment secret from **Hosts > Enroll Hosts** or the **osquery** overview. Orbit and direct osquery enrollment use the same secret.

Build the macOS package without embedding the Woodstar URL or enrollment secret:

```shell
fleetctl package --type=pkg --use-system-configuration
```

Deploy the package with the fleetd configuration profile below. Replace the example URL and secret before deployment.

```xml title="fleetd.mobileconfig"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>EnrollSecret</key>
      <string>REPLACE_WITH_SECRET</string>
      <key>FleetURL</key>
      <string>https://woodstar.example.com</string>
      <key>PayloadDisplayName</key>
      <string>fleetd</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.fleetd.config</string>
      <key>PayloadType</key>
      <string>com.fleetdm.fleetd.config</string>
      <key>PayloadUUID</key>
      <string>476F5334-D501-4768-9A31-1A18A4E1E807</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDescription</key>
  <string>Configures fleetd for Woodstar.</string>
  <key>PayloadDisplayName</key>
  <string>Woodstar - fleetd</string>
  <key>PayloadIdentifier</key>
  <string>com.example.woodstar.fleetd</string>
  <key>PayloadOrganization</key>
  <string>Example Organization</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

### Optional end-user mapping

Replace `$EMAIL` with the user-email variable supported by your MDM. Orbit reads the inner `com.fleetdm.fleet.mdm.apple.mdm` preference domain exactly as shown.

```xml title="fleetd-user.mobileconfig"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>EndUserEmail</key>
      <string>$EMAIL</string>
      <key>PayloadDisplayName</key>
      <string>fleetd user mapping</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.fleet.mdm.apple.mdm</string>
      <key>PayloadType</key>
      <string>com.fleetdm.fleet.mdm.apple.mdm</string>
      <key>PayloadUUID</key>
      <string>29713130-1602-4D27-90C9-B822A295E44E</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDescription</key>
  <string>Maps the assigned MDM user to a Woodstar host.</string>
  <key>PayloadDisplayName</key>
  <string>Woodstar - fleetd user</string>
  <key>PayloadIdentifier</key>
  <string>com.example.woodstar.fleetd-user</string>
  <key>PayloadOrganization</key>
  <string>Example Organization</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>9A11B43D-395C-4377-8EDB-870551531B41</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

See [Mutual TLS](./mutual-tls) to include a client certificate and key in the fleetd package.

## Orbit routes

| Method | Path                                    | Purpose                        |
| ------ | --------------------------------------- | ------------------------------ |
| `POST` | `/api/fleet/orbit/enroll`               | Enroll and return a node key   |
| `POST` | `/api/fleet/orbit/config`               | Return the Orbit configuration |
| `PUT`  | `/api/fleet/orbit/device_mapping`       | Record the assigned user email |
| `POST` | `/api/fleet/orbit/device_token`         | Rotate the device token        |
| `HEAD` | `/api/fleet/orbit/ping`                 | Check the server               |
| `HEAD` | `/api/latest/fleet/device/{token}/ping` | Validate a device token        |

## osquery routes

| Method | Path                                | Purpose                            |
| ------ | ----------------------------------- | ---------------------------------- |
| `POST` | `/api/v1/osquery/enroll`            | Enroll and return a node key       |
| `POST` | `/api/v1/osquery/config`            | Return schedule and client options |
| `POST` | `/api/v1/osquery/distributed/read`  | Return queued queries              |
| `POST` | `/api/v1/osquery/distributed/write` | Receive distributed-query results  |
| `POST` | `/api/v1/osquery/log`               | Receive scheduled report results   |

An invalid node key tells osquery to enroll again. Woodstar does not enable file carving or status-log forwarding.

Distributed results update host details, software inventory, dynamic labels, checks, reports, and active live queries.
