---
sidebar_position: 2
title: Processor Reference
description: The Woodstar AutoPkg processors and their inputs.
---

# Processor Reference

All four processors are published under `com.github.woodleighschool.woodstar.processors` and all of them need `WOODSTAR_URL` and `WOODSTAR_API_KEY`, usually set once in AutoPkg preferences (see [Overview](./overview#connection-settings)). Set `WOODSTAR_CA_FILE` when Woodstar uses a private CA.

## WoodstarMunkiAppUploader

Creates or updates the Woodstar software, its icon, and optionally its targets. Run it after `MunkiImporter`.

| Input          | Required | Notes                                                           |
| -------------- | -------- | --------------------------------------------------------------- |
| `name`         | no       | Munki item name. Defaults to the pkginfo `name`, then `NAME`.   |
| `display_name` | no       | Name shown to users. Defaults to the pkginfo `display_name`.    |
| `description`  | no       | Software description.                                           |
| `category`     | no       | Software category.                                              |
| `developer`    | no       | Software developer.                                             |
| `icon_path`    | no       | Icon file to upload and attach.                                 |
| `targets`      | no       | Full targets object with `include` and `exclude` label entries. |

It outputs `woodstar_software_id`, which the package processors pick up automatically.

## WoodstarMunkiPackageUploader

Converts the `MunkiImporter` pkginfo into Woodstar's package shape and creates or updates that version against the software. Run it after the app uploader.

| Input               | Required | Notes                                                                                      |
| ------------------- | -------- | ------------------------------------------------------------------------------------------ |
| `MUNKI_REPO`        | yes      | Repo path holding the `MunkiImporter` output.                                              |
| `pkginfo_repo_path` | no       | The pkginfo output path. Empty when `MunkiImporter` reused an existing item.               |
| `pkg_repo_path`     | no       | The package artifact path. Needed for package-bearing pkginfo.                             |
| `software_id`       | no       | Target software. Defaults to `woodstar_software_id` from the app uploader.                 |
| `eligible`          | no       | Whether this version counts when a target asks for the latest package. Defaults to `true`. |
| `force`             | no       | Upload the artifact even when Woodstar already has the same file. Defaults to `false`.     |

Package metadata keeps the Munki vocabulary it came with: installer type, restart action, blocking applications, `requires`, `update_for`, supported architectures, the unattended flags, and the rest. Relations use standard Munki item names such as `Dependency` or `Dependency-2.0`; every referenced item must exist in Woodstar.

## WoodstarMunkiPackageCleaner

Keeps the newest package versions for one software item and deletes the older ones. After removing package rows it tries to delete their installer artifacts; Woodstar keeps any artifact another package still references.

| Input                | Required | Notes                                                |
| -------------------- | -------- | ---------------------------------------------------- |
| `software_id`        | no       | Target software. Defaults to `woodstar_software_id`. |
| `keep_version_count` | no       | How many newest versions to keep. Defaults to `5`.   |

```yaml
Process:
    - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiPackageCleaner
      Arguments:
          keep_version_count: 5
```

## WoodstarMunkiRepoImporter

Imports every pkginfo under `MUNKI_REPO/pkgsinfo` in one pass, instead of going recipe by recipe. It creates missing software rows, leaves existing software metadata alone, and syncs package rows from pkginfo using the same conversion path as the package uploader. Software and package identities are created before relations are resolved, so references may point to an item that appears later in the repo. It does not set targets.

When the repo has duplicate pkginfo for the same software name and version, only the newest is imported, judged by the Munki metadata creation date and falling back to the pkginfo file's modification time.

| Input        | Required | Notes                                                                               |
| ------------ | -------- | ----------------------------------------------------------------------------------- |
| `MUNKI_REPO` | yes      | Repo path holding `pkgsinfo`, `pkgs`, and `icons`.                                  |
| `force`      | no       | Upload artifacts even when Woodstar already has the same file. Defaults to `false`. |

It has its own recipe, `com.github.woodleighschool.woodstar.munki.repo-import`. Run it like `makecatalogs`, after the recipes that produced the pkginfo:

```sh
autopkg run com.github.woodleighschool.munki.Blender \
  com.github.autopkg.munki.makecatalogs \
  com.github.woodleighschool.woodstar.munki.repo-import
```
