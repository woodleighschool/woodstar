---
sidebar_position: 4.5
title: Stemma
description: Publish software to Woodstar from a Stemma catalog.
---

# Stemma

The Woodstar plugin for [Stemma](https://woodleighschool.github.io/stemma/) publishes prepared installers and native Munki settings through the administrative API. This page covers what the plugin adds; Stemma's documentation covers catalogs, sources and commands.

## Connect

Create an [API key](../configuration/authentication#api-keys) whose account can edit Munki software and packages. Declare the plugin and a connection in the Stemma project:

```yaml
spec:
  plugins:
    woodstar:
      image: ghcr.io/woodleighschool/woodstar/stemma:TAG@sha256:DIGEST
  destinations:
    woodstar:
      operation: woodstar
      config:
        url: https://woodstar.example.com
        api_key: "{{ env.WOODSTAR_API_KEY }}"
```

Each release publishes the plugin at `ghcr.io/woodleighschool/woodstar/stemma`, tagged with the release; see [using plugins](https://woodleighschool.github.io/stemma/plugins) for pinning and updates. `url` is the server's HTTPS origin.

## Software settings

A software document sets the plugin's fields under its destination alias:

```yaml
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example-editor
spec:
  source:
    url: https://downloads.example.com/ExampleEditor.pkg
  destinations:
    woodstar:
      pkginfo:
        display_name: Example Editor
        unattended_install: true
      targets:
        include:
          - label_name: All Hosts
            actions:
              - optional_installs
              - managed_updates
      retention:
        keep: 2
```

`pkginfo` takes native Munki fields, including `RestartAction`, `receipts[].packageid`, `installs[].CFBundleIdentifier`, `items_to_copy`, scripts, alerts and installer environment variables. `name` defaults to the resource name. The plugin derives receipts, installed size, restart action, minimum OS, detection and removal settings from the prepared installer, and `items_to_copy` for an application in a DMG. Fields you set override derived values; a derived value that disappears clears its field.

Updates are sparse. Omitted fields stay unchanged, supplied lists replace their collections, and explicit `false`, `[]` and supported `null` values keep their meaning:

```yaml
pkginfo:
  description: null
  blocking_applications: []
  unattended_install: false
```

## Targets

Targets deploy software in place of repository catalogs. Write labels by their exact name; an unknown label fails before anything is written:

```yaml
targets:
  include:
    - label_name: All Hosts
      actions:
        - optional_installs
        - managed_updates
  exclude:
    - label_name: Exam MacBooks
```

Every include follows the software's latest package. See [Targets](munki#targets) for the available actions. Omitted target lists stay unchanged, each supplied `include` or `exclude` list replaces that collection, and setting both to `[]` leaves the software unassigned. New software without targets has no deployment labels.

## Relationships

`requires` and `update_for` entries name software by its Munki name, optionally followed by `--version` for a specific package, and resolve against software already in Woodstar. An entry can instead reference a catalog resource published to the same connection, which links through that resource's Munki name:

```yaml
pkginfo:
  requires:
    - EPSON Drivers
    - resource:
        kind: MacSoftware
        name: rosetta
```

Unknown or ambiguous names fail before anything is written, and `[]` clears the list. Stemma's [publication relationships](https://woodleighschool.github.io/stemma/publishing#publication-relationships) describe how it validates and orders referenced resources.

## Icons

The plugin publishes a resource's [declared icon](https://woodleighschool.github.io/stemma/mac-software#icons) as those exact bytes through the icon API. It creates a missing icon, replaces one whose content changed and keeps published artwork when the resource declares none. Icon changes never create a package version or upload the installer again.

## Items without an installer

A `nopkg` item publishes only scripts and metadata:

```yaml
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: managed-marker
spec:
  destinations:
    woodstar:
      pkginfo:
        version: "1.0"
        installer_type: nopkg
        installcheck_script: |
          #!/bin/sh
          test -f /Library/ManagedMarker && exit 1
          exit 0
        postinstall_script: |
          #!/bin/sh
          touch /Library/ManagedMarker
```

Munki runs the scripts on targeted clients; the plugin stores them without running them. The version identifies the package, so script or metadata edits at the same version update it in place.

## Identity and retention

The plugin keeps nothing between runs. Software names are unique and so is a version within its software, so it finds the declared title by exact name and its package by version, plans drift against them and converges on apply. Publishing over existing software takes it over; there is no separate adoption step. A package with the declared version but different installer bytes is drift and is replaced in place. An interrupted run is simply run again, and it uploads whatever the repository does not yet hold.

`retention.keep` applies to every package under the software, including versions published before Stemma. It keeps the declared version and the most recently created others up to the count. Packages pinned by a target or referenced by another package's `requires` or `update_for` stay protected and may exceed it. Cleanup runs after publication and targeting succeed.
