---
sidebar_position: 2
title: Processor Reference
description: Inputs and outputs for the Woodstar AutoPkg processors.
---

# Processor Reference

All processors require `WOODSTAR_URL` and `WOODSTAR_API_KEY`.

## WoodstarMunkiImporter

Generates pkginfo with Munki's `makepkginfo` and uploads the software, package, installer, optional icon, and targets.

| Input                            | Required | Description                                         |
| -------------------------------- | -------- | --------------------------------------------------- |
| `pkg_path`                       | yes      | Local package or disk image                         |
| `pkginfo`                        | no       | Values merged into the generated pkginfo            |
| `munkiimport_pkgname`            | no       | Value for `makepkginfo --pkgname`                   |
| `munkiimport_appname`            | no       | Value for `makepkginfo --appname`                   |
| `additional_makepkginfo_options` | no       | Additional `makepkginfo` arguments                  |
| `version_comparison_key`         | no       | Version key added to each generated `installs` item |
| `metadata_additions`             | no       | Values merged into pkginfo `_metadata`              |
| `icon_path`                      | no       | Prepared icon file                                  |
| `targets`                        | no       | Full include and exclude target object              |
| `force`                          | no       | Upload files even when unchanged; default `false`   |

The processor outputs `munki_info`, `woodstar_software`, `woodstar_software_id`, `woodstar_targets`, `woodstar_package`, and `woodstar_package_id`.

The following `MunkiImporter` inputs are not supported: `extract_icon`, `force_munkiimport`, `repo_subdirectory`, and `uninstaller_pkg_path`.

## Targets

An include accepts a `label_id` or `label_name`, a package selector, and one or more actions:

```yaml
targets:
    include:
        - label_name: Optional Apps
          package:
              strategy: latest
          actions:
              - optional_installs
              - managed_updates
    exclude:
        - label_name: Lab Macs
```

Package strategy is `latest` or `specific`. A specific package requires `package_id`.

Supported actions are `managed_installs`, `managed_uninstalls`, `managed_updates`, `optional_installs`, `featured_items`, and `default_installs`.

## WoodstarMunkiPackageCleaner

Deletes older package versions for one software title.

| Input                | Required | Description                                            |
| -------------------- | -------- | ------------------------------------------------------ |
| `software_id`        | no       | Software ID; defaults to `woodstar_software_id`        |
| `keep_version_count` | no       | Number of newest package versions to keep; default `5` |

The processor outputs `woodstar_deleted_package_count`.

## WoodstarMunkiRepoImporter

Imports an existing Munki repository. The processor creates missing software titles, imports icons for new titles, and creates or updates packages from pkginfo. Existing software metadata and targets are left unchanged.

| Input        | Required | Description                                               |
| ------------ | -------- | --------------------------------------------------------- |
| `MUNKI_REPO` | yes      | Repository containing `pkgsinfo`, `pkgs`, and `icons`     |
| `force`      | no       | Upload package files even when unchanged; default `false` |

When several pkginfo files have the same name and version, the importer selects the newest by Munki creation date, then file modification time. Package relationships may refer to items that appear later in the repository.

The bundled recipe identifier is `com.github.woodleighschool.woodstar.munki.repo-import`.
