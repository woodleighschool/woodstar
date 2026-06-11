# Woodstar AutoPkg Processors

This contains processors that can be used in AutoPkg to upload MunkiImporter
output to Woodstar and optionally set Munki software targets.

## Setup

Configure the Woodstar connection and local Munki repo in AutoPkg prefs:

```sh
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
defaults write com.github.autopkg MUNKI_REPO -string "/Users/Shared/munki_repo"
defaults write com.github.autopkg WOODSTAR_URL -string "http://localhost:8080"
```

## Recipe Flow

Run the normal Munki import first. `WoodstarMunkiAppUploader` creates or updates
the Woodstar Munki software using the pkginfo `display_name` as the name when it
is present. `WoodstarMunkiPackageUploader` then converts the generated pkginfo
into Woodstar's normal package API shape and creates or updates the matching
package version using `woodstar_software_id`.

Package artifacts are reused by default when Woodstar already has the same
filename, SHA-256, and size at the expected storage location. Pass `-k force=true`
to upload the package artifact again. App metadata and package metadata are
cheap upserts; AutoPkg summaries are only emitted when Woodstar changes.

`requires` and `update_for` must contain Woodstar package IDs, matching what
Woodstar will render as Munki item names. `nopkg` items are imported without a
package artifact. `uninstall_package` items upload `uninstaller_pkg_path` as a
separate uninstaller artifact.

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

`targets.include` is ordered from highest to lowest priority for hosts that
match multiple labels. Each include entry accepts `label_id` or `label_name`, a
package selector, and an action list:

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

## Cleanup

`WoodstarMunkiPackageCleaner` keeps the newest package rows for one software item
and deletes older rows. It defaults to `woodstar_software_id` from the app
uploader and keeps 5 versions:

```yaml
Process:
  - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiPackageCleaner
    Arguments:
      keep_version_count: 5
```

After deleting package rows it tries to delete their installer artifacts. Woodstar
will keep artifacts that are still referenced by another package.

## Repo Import

`com.github.woodleighschool.woodstar.munki.repo-import` imports every pkginfo
under `MUNKI_REPO/pkgsinfo`. Use it as a recipe after the app recipes in the
same run, similar to where `com.github.autopkg.munki.makecatalogs` sits in a
recipe list:

```sh
autopkg run com.github.woodleighschool.munki.Blender \
  com.github.autopkg.munki.makecatalogs \
  com.github.woodleighschool.woodstar.munki.repo-import
```

It creates missing software rows, leaves existing software metadata alone, and
syncs package rows from pkginfo. It uses the same package conversion path as the
per-recipe package processor and does not set targets.
