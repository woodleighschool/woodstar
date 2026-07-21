---
sidebar_position: 1
title: Overview
description: How AutoPkg recipes push packages into Woodstar's Munki repo.
---

# AutoPkg

[AutoPkg](https://github.com/autopkg/autopkg) is how packages get into Woodstar without anyone hand-building pkginfo. You run a normal Munki recipe, and a few extra processors take the output, create or update the matching Woodstar software and package, and optionally set where it deploys.

Woodstar ships those processors in the repo, under `autopkg/`. The [processor reference](./processors) lists each one and its inputs.

## Getting AutoPkg to see the processors

Add the Woodstar repo to AutoPkg so it can find the processors and the bundled recipes:

```sh
autopkg repo-add woodleighschool/woodstar
```

The processors are published under the identifier `com.github.woodleighschool.woodstar.processors`, which is how recipes refer to them.

## Connection settings

The processors read Woodstar's URL and an admin API key from AutoPkg preferences, alongside the usual Munki repo path:

```sh
defaults write com.github.autopkg WOODSTAR_URL -string "https://woodstar.example"
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
defaults write com.github.autopkg MUNKI_REPO -string "/Users/Shared/munki_repo"
```

The API key is an account API key from Woodstar; see [Authentication](../configuration/authentication#api-keys) for where those come from.

For a private or development CA, set its PEM file explicitly. The same CA is used for API requests and direct artifact uploads:

```sh
defaults write com.github.autopkg WOODSTAR_CA_FILE -string "/path/to/woodstar-ca.pem"
```

## The recipe flow

Run the normal Munki import first, then the Woodstar processors:

```yaml
Process:
    - Processor: MunkiImporter

    - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiAppUploader
      Arguments:
          targets:
              include:
                  - label_name: All Hosts
                    package:
                        strategy: latest
                    actions:
                        - managed_installs
              exclude: []

    - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiPackageUploader
```

`WoodstarMunkiAppUploader` creates or updates the Woodstar software using the pkginfo `name`, Munki's stable item identity. `display_name` remains presentation metadata. `WoodstarMunkiPackageUploader` then turns the generated pkginfo into Woodstar's package shape and creates or updates the matching version against that software.

Both are cheap upserts. AutoPkg only prints a summary line when Woodstar actually changed something.

## Package artifacts

A package artifact is reused when Woodstar already has the same filename, SHA-256, and size at the expected storage location, so re-running a recipe doesn't re-upload bytes that haven't changed. Pass `-k force=true` to upload again anyway.

A few pkginfo shapes carry through as you'd expect:

- `requires` and `update_for` use normal Munki item names, optionally followed by `-version`. The uploader resolves them to Woodstar software and package references.
- `nopkg` items are imported with no package artifact.

## Targets

`targets.include` is ordered from highest to lowest priority, which decides the outcome for a host that matches more than one label. Each include entry takes a `label_id` or `label_name`, a package selector, and a list of actions:

```yaml
targets:
    include:
        - label_name: Optional Apps
          package:
              strategy: specific
              package_id: 123
          actions:
              - managed_updates
              - optional_installs
              - featured_items
    exclude:
        - label_name: Excluded Devices
```

The action list is preserved as written. Combining `optional_installs` with `managed_updates` makes the item available for an initial user install and makes later updates mandatory once any version is installed.

This is the same Munki [target model](../admin/munki#targets), set from the recipe instead of the admin app.

## Importing an existing repo

If you already have a Munki repo full of pkginfo, the repo importer brings it in wholesale rather than recipe by recipe. See [the processor reference](./processors#woodstarmunkirepoimporter).
