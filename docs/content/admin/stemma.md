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
Authored native values take precedence. Use `stemma inspect FILE` to examine
application evidence when a source needs an explicit selection.

`pkginfo` uses supported native Munki fields, including `RestartAction`,
`receipts[].packageid`, `installs[].CFBundleIdentifier`, `items_to_copy`, scripts,
alerts, and installer environment variables. Typed `$fact` references retain their
native value type. A JSON pkginfo artifact can also be supplied with its installer
through `inputs.installer`.

## Application icons

`MacSoftware` automatically prepares an `icon` PNG when a selected application
provides one. macOS uses the native system renderer for current system styling;
Linux and Windows use supported portable application icon resources. A catalog
can therefore bootstrap in CI without icon-specific configuration. Native rendering
is an enhancement; unsupported applications and installers can remain iconless.
A PKG without a selected application directory is not rendered as an app icon.

The plugin creates a missing icon independently of software or package creation.
It retains existing artwork across normal runs and software updates, including runs
that provide no icon. A later portable run does not replace a native icon.
To intentionally improve artwork across the catalog, run on a current Mac:

```sh
stemma apply --refresh-icons
stemma apply MacSoftware/example-editor-amd64 --refresh-icons
```

`stemma plan --refresh-icons` previews the icon changes. Refresh reuses installer
preparation and changes only icon content when the software is otherwise current.
The plugin uploads and attaches a new storage object through the existing icon API;
it does not create another package version or reupload the installer.

## Targets and native settings

Targets belong to the destination alongside `pkginfo`. To offer the latest package and update installed copies on a label, add:

```yaml
targets:
  include:
    - label_id: 123
      package:
        strategy: latest
      actions:
        - optional_installs
        - managed_updates
  exclude: []
```

Use `strategy: specific` with an existing `package_id` to pin a version. See [Targets](munki#targets) for the available actions. Omitted target lists remain unmanaged; an empty list clears that list. A newly created software title without targets has no deployment labels.

The shared importer converts native fields to sparse API PATCH requests. Omitted native fields remain unmanaged, supplied lists replace their collections, and explicit `false`, empty lists, and supported `null` clears retain their meaning. For example:

```yaml
pkginfo:
  description: null
  blocking_applications: []
  unattended_install: false
```

Previously derived optional fields clear when the corresponding evidence disappears. To suppress derivation and relinquish ownership of a field, list it under `unmanaged`, such as `pkginfo.minimum_os_version`; that field cannot also be authored in `pkginfo`.

Native `requires` and `update_for` entries identify software by its Munki name, optionally followed by `--version` for a specific package. Unknown or ambiguous references fail before writing; `[]` clears the relationship list. Deployment uses targets in place of repository catalogs.

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
      targets:
        include: []
        exclude: []
      retention:
        keep: 1
```

Munki evaluates the scripts on targeted clients according to its installation checks. Stemma stores them without executing them. The native version identifies a `nopkg` publication; script or metadata edits at the same version update its existing package.

## Publish and retain versions

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

Stages and diagnostics go to stderr, including the plugin's upload, publication, and verification stages. Terminals show live progress; CI and redirected output use ordinary lines. `--verbose` (`-v`) enables debug diagnostics, `--quiet` (`-q`) keeps warnings and errors, and `--log-level debug|info|warn|error` selects an explicit threshold. `--no-progress` disables animation. Use `--output json` for the final stdout report and `--log-format json` for structured stderr logs. Log levels leave reports intact.

Generate a project editor schema with `stemma schema --project --offline` for field validation and help from the configured plugins.

`retention.keep` retains the current publication plus the most recent successfully published distinct payloads up to the requested count. Pinned or referenced packages remain protected and may exceed that count. Cleanup runs after publication and targeting succeed, and only deletes package IDs recorded as owned with a durable publication order. Metadata-only changes do not advance that order.

Commit the configuration and reviewed lockfile. Preserve `.stemma/state` between runs, or set `STEMMA_STATE_DIR` to a persistent directory; the content cache is disposable. An existing software name or package version does not establish ownership when a binding is missing. Restore the durable binding before publishing again. To update the plugin, change its image reference, run `stemma plugins update`, and review the lockfile.
