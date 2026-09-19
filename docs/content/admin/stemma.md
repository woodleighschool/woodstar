---
sidebar_position: 4.5
title: Stemma
description: Acquire, prepare, and publish managed software with Stemma.
---

# Stemma

[Stemma](https://github.com/woodleighschool/stemma) acquires and inspects software without executing installer payloads. Our standalone `woodstar.munki` plugin publishes installers and native Munki settings through the administrative API. Stemma loads it from its advertised operation contract.

## Project and software families

Create a Git repository for the configuration and an [API key](../configuration/authentication#api-keys) with permission to edit Munki software and packages. Supply the key through `WOODSTAR_API_KEY` and the chosen plugin image tag or digest through `WOODSTAR_PLUGIN_IMAGE`.

Save the root project as `stemma.yaml`:

```yaml
apiVersion: stemma/v1alpha1
kind: Project
metadata:
  name: managed-apps
spec:
  imports:
    - software/*.yaml
  plugins:
    woodstar:
      trusted: true
      image: ${WOODSTAR_PLUGIN_IMAGE}
  destinations:
    woodstar:
      operation: woodstar.munki
      config:
        url: https://woodstar.example.com
        api_key: ${WOODSTAR_API_KEY}
```

The plugin image uses the OCI reference `ghcr.io/woodleighschool/woodstar/stemma` with a tag or digest. Stemma selects a bundle for the runner independently of the target software's architecture. For a private CA, add `ca_file` with an absolute PEM certificate path to the connection config.

Imported family files contain one or more `MacSoftware` documents separated by `---`. Each document has its own stable identity. For example, `software/example-editor.yaml` can publish separate architecture variants:

```yaml
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example-editor-arm64
spec:
  source:
    url: https://downloads.example.com/ExampleEditor-1.2.3-arm64.pkg
    filename: ExampleEditor.pkg
  destinations:
    woodstar:
      pkginfo:
        display_name: Example Editor
        description: Managed application for Apple silicon
        version: 1.2.3
        supported_architectures:
          - arm64
        unattended_install: true
      retention:
        keep: 2
---
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example-editor-amd64
spec:
  source:
    url: https://downloads.example.com/ExampleEditor-1.2.3-x86_64.dmg
    filename: ExampleEditor.dmg
  application:
    path: ExampleEditor.app
    installed_path: /Applications/ExampleEditor.app
  destinations:
    woodstar:
      pkginfo:
        display_name: Example Editor
        supported_architectures:
          - x86_64
      retention:
        keep: 2
```

`MacSoftware` supplies the prepared installer and selected application evidence.
Native `pkginfo.name` defaults to the resource's identity, keeping architecture
variants separate in Munki. `installer` is only needed to select another named
resource output.

For PKGs, inspection supplies payload receipts and an unambiguous version. An app
selection supplies its version, minimum OS, and detection facts. For a DMG app,
the plugin derives `items_to_copy` and detection at the selected installed path.
Native values you set take precedence. Use `stemma inspect FILE` to examine
application evidence when a source needs an explicit selection.

`pkginfo` uses supported native Munki fields, including `RestartAction`,
`receipts[].packageid`, `installs[].CFBundleIdentifier`, `items_to_copy`, scripts,
alerts, and installer environment variables. Typed `$fact` references retain their
native value type.

## Icons

A resource that declares `spec.icon` supplies the committed PNG from the catalog's
`icons/` directory. Create it from the software with `stemma icon`, or commit your
own artwork for software that carries none:

```yaml
spec:
  icon: example-editor
```

The plugin publishes those exact bytes. It creates a missing icon independently of
software or package creation, replaces the icon when the file's content changes and
keeps published artwork when the resource declares none. It uploads and
attaches a storage object through the existing icon API; it does not create another
package version or reupload the installer.

## Targets and native settings

Targets belong to the destination alongside `pkginfo`. To offer the latest package and update installed copies on a label, add:

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

Labels are written by their exact name; an unknown label fails before writing. Every include follows the software's latest package. See [Targets](munki#targets) for the available actions. Omitted target lists stay unchanged. Each supplied `include` or `exclude` list replaces that collection; setting both to `[]` leaves an item unassigned. A newly created software title without targets has no deployment labels.

The shared importer converts native fields to sparse API PATCH requests. Omitted native fields are left unchanged, supplied lists replace their collections, and explicit `false`, empty lists, and supported `null` clears retain their meaning. For example:

```yaml
pkginfo:
  description: null
  blocking_applications: []
  unattended_install: false
```

Explicit fields override derived values. Installer evidence supplies receipts, installed size, restart action, DMG copy details, minimum OS, detection and removal settings. Missing derived values clear those fields; other omitted fields stay unchanged.

Native `requires` and `update_for` entries identify software by its Munki name, optionally followed by `--version` for a specific package, and resolve against software already in Woodstar. An entry may instead name a catalog resource published to the same connection; it links through the Munki name that resource declares, its declared `pkginfo.name` or else its own name, and Stemma reconciles the resource first:

```yaml
pkginfo:
  requires:
    - software: rosetta
    - software: microsoft-365-business-pro-suite
      version: "16.113"
    - EPSON Drivers
```

Unknown or ambiguous references fail before writing; `[]` clears the relationship list. Deployment uses targets in place of repository catalogs.

## Source-free items

A `nopkg` item publishes scripts and metadata without a source or installer. Add another MacSoftware document to an imported family file:

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
        display_name: Managed marker
        installcheck_script: |
          #!/bin/sh
          test -f /Library/ManagedMarker && exit 1
          exit 0
        postinstall_script: |
          #!/bin/sh
          touch /Library/ManagedMarker
      retention:
        keep: 1
```

Munki evaluates the scripts on targeted clients according to its installation checks. Stemma stores them without executing them. The native version identifies a `nopkg` publication; script or metadata edits at the same version update its existing package.

## Publish

Run from the project directory:

```sh
stemma plugins install
stemma operations --offline
stemma validate --offline
stemma update
stemma prepare
stemma plan
stemma apply
```

`plugins install` locks the release index and fetches the runner's bundle. `validate` checks configuration and plugin contracts before acquisition. `prepare` records new inputs and local changes, reuses existing remote pins, and prepares installers. `update` explicitly refreshes upstream sources. `plan` and `apply` require reviewed locks; `plan` reads remote state, and `apply` publishes changed content, metadata, and supplied targets. Metadata edits reuse cached acquisition and installer preparation.

Stages and diagnostics go to stderr, including the plugin's upload, publication, and verification stages. Terminals show live progress and uploaded bytes; CI and redirected output use ordinary lines. `--verbose` (`-v`) enables debug diagnostics, `--quiet` (`-q`) keeps warnings and errors, and `--log-level debug|info|warn|error` selects an explicit threshold. `--no-progress` disables animation. Use `--json` for the final stdout report and `--log-format json` for structured stderr logs. Log levels leave reports intact.

Generate a project editor schema with `stemma schema --project --offline` for field validation and help from the configured plugins.

## Identity and retention

The plugin keeps nothing between runs. Software names are unique and so is a version within its software, so it finds the declared title by exact name and its package by version, plans ordinary drift against them and converges on `apply`. Publishing over software that already exists is how a catalog takes it on: there is no separate adoption step. A package with the declared version but different installer bytes is drift, reported by `plan` and replaced in place. An interrupted run is simply run again; it uploads whatever the repository does not yet hold.

`retention.keep` applies to every package under the software, including versions published before Stemma. It retains the declared version plus the most recently created others up to the requested count. Packages pinned by a target or referenced by another package's `requires` or `update_for` remain protected and may exceed that count. Cleanup runs after publication and targeting succeed.

Commit the configuration and reviewed lockfile; the content cache is disposable and there is no other local state. To update the plugin, change its image reference, run `stemma plugins update`, and review the lockfile.
