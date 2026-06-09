# Woodstar AutoPkg Processors

This contains processors that can be used in AutoPkg to upload a pkginfo to Woodstar
and optionally set Munki software targets.

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
is present. `WoodstarMunkiPackageUploader` then imports the pkginfo into that
software using `woodstar_software_id`.

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
