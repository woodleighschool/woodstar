---
sidebar_position: 1
title: Overview
description: Import Munki packages into Woodstar with AutoPkg.
---

# AutoPkg

Woodstar includes AutoPkg processors for importing Munki software and package versions. The normal recipe flow uploads directly to Woodstar and does not require a Munki repository.

## Install the processors

```sh
autopkg repo-add woodleighschool/woodstar
```

Recipes refer to them through `com.github.woodleighschool.woodstar.processors`.

## Configure access

Set the Woodstar URL and an account API key in AutoPkg preferences:

```sh
defaults write com.github.autopkg WOODSTAR_URL -string "https://woodstar.example"
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
```

## Import a package

Use `WoodstarMunkiImporter` after a download processor has produced `pkg_path`:

```yaml
Process:
    - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiImporter
      Arguments:
          pkg_path: "%pkg_path%"
          icon_path: "%RECIPE_CACHE_DIR%/%NAME%.png"
          pkginfo:
              name: "%NAME%"
              display_name: "%NAME%"
              catalogs:
                  - testing
          targets:
              include:
                  - label_name: All Hosts
                    package:
                        strategy: latest
                    actions:
                        - managed_installs
              exclude: []
```

The processor runs `/usr/local/munki/makepkginfo`, creates or updates the software title and package version, and uploads the installer and optional icon. Re-running the recipe skips unchanged records and files. Use `force=true` to upload the files again.

Targets can use `label_id` or `label_name`. Includes are evaluated in their listed order. If `targets` is omitted, existing targets are retained and new software is created without targets.

## Clean old versions

Run `WoodstarMunkiPackageCleaner` after the importer:

```yaml
- Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiPackageCleaner
  Arguments:
      keep_version_count: 5
```

`MunkiPackageRemover` uses `woodstar_software_id` from the importer and deletes older package versions.

## Import an existing Munki repository

`WoodstarMunkiRepoImporter` imports every pkginfo under `MUNKI_REPO/pkgsinfo` with its package and icon files:

```sh
autopkg run com.github.woodleighschool.woodstar.munki.repo-import
```

The repository importer does not set software targets. See [Processor Reference](./processors) for its behaviour and inputs.
