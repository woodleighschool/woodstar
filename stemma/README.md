# Stemma plugin 🌿

Publish installers and native Munki settings through the administrative API.

## 🚀 Usage

See the [Stemma guide](https://woodleighschool.github.io/woodstar/docs/admin/stemma) for configuration, acquisition, and publication.

The plugin exposes `woodstar.munki`. MacSoftware supplies the installer,
application evidence and optional icon. Destination settings contain native
`pkginfo`, deployment `targets` and `retention.keep`. Source-free `nopkg` items
publish scripts and metadata.

Software names and package versions identify remote objects; the plugin keeps no
local publication state. Explicit settings override derived values, and other
omitted fields remain unchanged. Retention preserves pinned or referenced packages.
Icons update independently of installers.

## 🧑‍💻 Development

Run from this directory:

```sh
mise install
mise run build
mise run test
mise run lint
```

The binary is written to `build/plugin`. This directory is a separate Go
module using the server models from the parent checkout and Stemma's public
`plugin` SDK. Use `mise run test-postgres` for the PostgreSQL integration test.

## 📦 Packaging

GoReleaser builds static `plugin` executables (`plugin.exe` on Windows) and
deterministic tar.zst bundles for Darwin, Linux, and Windows on amd64 and arm64.
Each bundle contains the executable and licence at its root.

The release workflow publishes the bundles with Stemma's publish-plugin action as
one OCI platform index at `ghcr.io/woodleighschool/woodstar/stemma`, tagged with
the release.

Run `mise run snapshot` to build all release bundles locally.
