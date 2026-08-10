---
sidebar_position: 3
title: Santa Sync
description: Configure Santa to sync with Woodstar.
---

# Santa Sync

Woodstar implements Santa's SyncV1 protocol over HTTPS with gzipped protobuf messages.

## Configure Santa

Create a secret from the **Santa** overview, then replace the example URL and secret in this profile:

```xml title="santa.mobileconfig"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>ClientMode</key>
      <integer>1</integer>
      <key>SyncBaseURL</key>
      <string>https://woodstar.example.com/santa/sync/</string>
      <key>SyncClientContentEncoding</key>
      <string>gzip</string>
      <key>SyncEnableProtoTransfer</key>
      <true/>
      <key>SyncExtraHeaders</key>
      <dict>
        <key>Authorization</key>
        <string>Bearer REPLACE_WITH_SECRET</string>
      </dict>
      <key>PayloadDisplayName</key>
      <string>Santa</string>
      <key>PayloadIdentifier</key>
      <string>com.northpolesec.santa.4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA</string>
      <key>PayloadType</key>
      <string>com.northpolesec.santa</string>
      <key>PayloadUUID</key>
      <string>4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDescription</key>
  <string>Configures Santa for Woodstar.</string>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Santa</string>
  <key>PayloadIdentifier</key>
  <string>com.example.woodstar.santa</string>
  <key>PayloadOrganization</key>
  <string>Example Organization</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>7CE340DE-AAB6-448B-A558-EB3C49A3A687</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

The profile sets:

- `SyncBaseURL` to `<WOODSTAR_URL>/santa/sync/`
- Protobuf transfer and gzip encoding
- The Santa secret in the `Authorization` header

Leave `MachineID` unset so Santa uses the hardware UUID also reported by Orbit or osquery.

See [Mutual TLS](./mutual-tls#santa) for Santa's PKCS#12 and System Keychain options.

## Sync routes

| Method | Path                                    | Purpose                                 |
| ------ | --------------------------------------- | --------------------------------------- |
| `POST` | `/santa/sync/preflight/{machine_id}`    | Record client state and prepare rules   |
| `POST` | `/santa/sync/eventupload/{machine_id}`  | Upload execution and file-access events |
| `POST` | `/santa/sync/ruledownload/{machine_id}` | Download rules                          |
| `POST` | `/santa/sync/postflight/{machine_id}`   | Confirm the applied rule set            |

Santa must match an existing host. Preflight updates the Santa version, client mode, user details, SIP state, and rule-sync state. Rule downloads are paged in batches of 500.

Woodstar supports SyncV1. Santa SyncV2 and the associated push infrastructure are not implemented.

See [Santa](../admin/santa) for configurations, rules, and events.
