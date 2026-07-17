# Woodstar AutoPkg Processors

Woodstar's AutoPkg processor generates Munki pkginfo and uploads the resulting software, package, installer, optional icon, and targets directly to Woodstar. The normal recipe flow does not use a Munki repository or AutoPkg's `MunkiImporter`.

Make them available to AutoPkg:

```sh
autopkg repo-add woodleighschool/woodstar
```

The processors are published under `com.github.woodleighschool.woodstar.processors`.

Set the Woodstar connection once in AutoPkg preferences:

```sh
defaults write com.github.autopkg WOODSTAR_URL -string "https://woodstar.example"
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
```

Then replace the usual `MunkiImporter` and Woodstar uploader chain with one processor:

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
          - label_name: Optional Apps
            package:
              strategy: latest
            actions:
              - optional_installs
              - managed_updates
        exclude: []
```

`pkg_path` is inspected with Munki's installed `/usr/local/munki/makepkginfo` and uploaded from its existing local path. `icon_path` is optional and must point to an already prepared image. The processor also accepts `munkiimport_pkgname`, `munkiimport_appname`, `additional_makepkginfo_options`, `version_comparison_key`, and `metadata_additions` with the same generation meaning as AutoPkg's core importer.

The direct importer does not extract icons and does not accept `uninstaller_pkg_path`; Woodstar currently has one installer artifact per package and cannot represent a separate uninstaller package.

`WoodstarMunkiPackageCleaner` removes older Woodstar package versions after an import. `WoodstarMunkiRepoImporter` remains a separate migration processor for loading an existing `MUNKI_REPO`; repository paths are not part of the normal recipe contract.
