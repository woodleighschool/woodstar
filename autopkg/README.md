# Woodstar AutoPkg processors

The Woodstar processors create Munki pkginfo and upload software, packages, installers, icons, and targets to Woodstar.

Add the repository to AutoPkg:

```sh
autopkg repo-add woodleighschool/woodstar
```

Set the Woodstar URL and an account API key:

```sh
defaults write com.github.autopkg WOODSTAR_URL -string "https://woodstar.example"
defaults write com.github.autopkg WOODSTAR_API_KEY -string "API_KEY"
```

## Import a package

`WoodstarMunkiImporter` runs Munki's installed `/usr/local/munki/makepkginfo`, then uploads the result directly to Woodstar. No local Munki repository is required.

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

`icon_path` and `targets` are optional. Omitting `targets` preserves them on existing software and leaves new software untargeted.

The importer does not extract icons or accept a separate `uninstaller_pkg_path`.

## Other processors

- `WoodstarMunkiPackageCleaner` keeps the newest package versions for a software title.
- `WoodstarMunkiRepoImporter` imports pkginfo, packages, and icons from an existing `MUNKI_REPO`.

See the [AutoPkg documentation](https://woodleighschool.github.io/woodstar/docs/autopkg/overview) for examples and the full input reference.
