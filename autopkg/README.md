# Woodstar AutoPkg Processors

This contains processors that can be used in AutoPkg to upload a pkginfo to Woodstar
and optionally create Munki software assignments.

## Setup

Configure the Woodstar connection and local Munki repo in AutoPkg prefs:

```sh
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
defaults write com.github.autopkg MUNKI_REPO -string "/Users/Shared/munki_repo"
defaults write com.github.autopkg WOODSTAR_URL -string "http://localhost:8080"
```

## Recipe Flow

Run the normal Munki import first. `WoodstarMunkiPackageUploader` expects the
environment produced by `MunkiImporter`, including pkginfo and package paths.
Exclude labels can be supplied as `exclude_label_ids` or `exclude_label_names`.

```yaml
Process:
  - Processor: MunkiImporter

  - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiAppUploader
    Arguments:
      assignments:
        includes:
          - priority: 1
            label_name: All Hosts
            action: install
            package_selection: latest_eligible
        exclude_label_ids: []

  - Processor: com.github.woodleighschool.woodstar.processors/WoodstarMunkiPackageUploader
```
