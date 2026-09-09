---
sidebar_position: 4.5
title: Stemma
description: Acquire, prepare, and publish managed software with Stemma.
---

# Stemma

[Stemma](https://github.com/woodleighschool/stemma) downloads software, prepares installers, and renders native Munki pkginfo. The `woodstar.munki` operation publishes that document and its installer through the administrative API.

## Set up a project

Download [Stemma](https://github.com/woodleighschool/stemma/releases), put `stemma` on your `PATH`, and create a Git repository for the configuration.

Create an [API key](../configuration/authentication#api-keys) using an account with permission to edit Munki software and packages. Supply it through the `WOODSTAR_API_KEY` environment variable or your CI secret settings; keep the key out of YAML and Git.

Save the following as `stemma.yaml`, replacing the release tag, HTTPS origin, installer source, and rollout label with your values. Stemma selects the plugin bundle for the runner, independently of the software's target architecture. For a private CA, add `ca_file: /absolute/path/woodstar-ca.pem` under the destination's `config`.

```yaml
version: 1
project: managed-apps
plugins:
  woodstar:
    trusted: true
    image: ghcr.io/woodleighschool/woodstar/stemma:vX.Y.Z
destinations:
  woodstar:
    operation: woodstar.munki
    config:
      url: https://woodstar.example.com
      api_key: ${WOODSTAR_API_KEY}
recipes:
  example-app:
    source:
      type: http
      url: https://downloads.example.com/ExampleApp-1.2.3.pkg
      filename: ExampleApp.pkg
    platform: darwin
    arch: universal
    steps:
      - name: pkginfo
        operation: munki.pkginfo
        inputs:
          input: prepared
        config:
          name: ExampleApp
          version: 1.2.3
          display_name: Example App
          description: Managed application
          unattended_install: true
    destinations:
      woodstar:
        artifact: pkginfo/artifact
        inputs:
          installer: prepared
        targets:
          include:
            - label_id: 123
              package:
                strategy: latest
              actions:
                - optional_installs
                - managed_updates
```

The render step produces a native JSON pkginfo document. Its configuration uses Munki keys such as `RestartAction`, `receipts[].packageid`, and `installs[].CFBundleIdentifier`. Stemma derives detection metadata from inspected artifacts; explicitly supplied fields take precedence. Use `stemma inspect /path/to/ExampleApp.pkg` to review versioned subjects, receipt facts, and application facts. If observed versions are ambiguous, select a version explicitly or use Stemma's typed `$fact` references to a named subject.

The destination selects the pkginfo document as its primary artifact and the matching installer as a named input. The plugin verifies both files and checks the pkginfo installer hash before publication. It uses the shared Munki importer to convert the document into sparse API mutations; package locations and upload object IDs remain owned by storage.

Targets control deployment. The example offers the latest version in Managed Software Center and updates installed copies on matching Macs. Use `strategy: latest` for first publication. To pin an existing version, use `strategy: specific` with a `package_id` belonging to the software title. See [Targets](munki#targets) for available actions.

## Acquire, prepare, and publish

Run from the project directory:

```sh
stemma plugins install
stemma operations
stemma validate
stemma update
stemma prepare --frozen-lockfile
stemma plan --frozen-lockfile
stemma apply --frozen-lockfile
```

`plugins install` locks the OCI release index and fetches only the runner’s bundle. `operations` lists built-in and external operations with their contracts; operation names must be unique across providers. `validate` checks configuration and trusted plugin contracts before acquiring recipe inputs. `update` resolves source inputs and writes their reviewed identities and hashes to `stemma.lock.yaml`.

`prepare` inspects inputs and runs the configured steps without publishing or executing installer payloads. `plan` reads the remote state and reports changes. `apply` uploads changed installer content and reconciles software, package metadata, and supplied targets. Repeating the same configuration converges on the existing title and version. A missing local binding recovers the same remote objects by exact software name and package version.

For a software update, change the source and review the intended native version, run `stemma update`, then repeat preparation, planning, and application. For a metadata change, run `plan` and `apply` again; only the affected render step needs new output. Acquisition and installer preparation remain cached independently of publication metadata.

Commit the configuration and reviewed lockfile. Preserve `.stemma/state` between runs, or set `STEMMA_STATE_DIR` to a persistent directory. The content cache is disposable. To update the plugin, change its image tag, run `stemma plugins update`, and review the new lock.

## Manage existing settings

Omitted fields remain unmanaged except for Stemma's derived native defaults. Objects merge recursively, supplied lists replace their collections, and explicit `false`, empty lists, and supported `null` clears retain their meaning.

For example, these fields in the `munki.pkginfo` step's `config` clear the description and blocking applications, and disable unattended installation:

```yaml
description: null
blocking_applications: []
unattended_install: false
```

Keep the step's required `name` and effective version. To clear only target exclusions, set `targets.exclude` to `[]` in the recipe's destination settings; the omitted include list stays unchanged. Targets are the plugin's only publication metadata control. Native software and package settings belong in the render step.

Native `requires` lists identify existing software by name, optionally followed by `--version` for a specific package. The plugin resolves these references to API IDs; an unknown or ambiguous reference fails before writing. Use `[]` to clear a relationship list. Catalogs and icon files require separate repository management and cannot be supplied through this transport.
