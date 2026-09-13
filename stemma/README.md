# Stemma plugin 🌿

Publish installers and native Munki settings through the administrative API.

## 🚀 Usage

See the [Stemma guide](https://woodleighschool.github.io/woodstar/docs/admin/stemma) for configuration, acquisition, and publication.

The standalone plugin exposes `woodstar.munki` through Stemma's operation protocol.
MacSoftware supplies its installer, selected application evidence, and optional icon.
Author native `pkginfo`, deployment `targets`, and `retention.keep` on the destination. Source-free `nopkg`
items publish scripts and metadata without an installer.

The shared server importer maps native fields to sparse PATCH requests. Supplied
values remain explicit; omitted fields stay unmanaged unless owned by derivation.
Durable bindings track published payloads and protect against adopting unrelated
software by name. Retention removes only owned package versions after publication
succeeds and preserves pinned or referenced packages.

Application icons are created when missing and retained on normal runs, even if
later preparation has no icon. Run `stemma apply --refresh-icons` on macOS to
replace artwork using the native renderer. Resource selection composes with the
flag; `stemma plan --refresh-icons` previews changes. Icon publication does not
reupload installers or create package versions.

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
`plugin` SDK. The operation advertises `ConfigSchema`, `MetadataSchema`, and
`RequiresInspection`; Stemma discovers the contract through the trusted plugin.
Use `mise run test-postgres` for the PostgreSQL integration test.

## 📦 Packaging

GoReleaser builds static `plugin` executables (`plugin.exe` on Windows) and
deterministic tar.zst bundles for Darwin, Linux, and Windows on amd64 and arm64.
Each bundle contains the executable and licence at its root.

ORAS publishes the bundles to `ghcr.io/woodleighschool/woodstar/stemma` using
Stemma’s OCI artifact contract. The release workflow publishes one platform index
under the release tag after every bundle succeeds. Registry authentication uses
the standard credential store; no container runtime is required.

Run `mise run snapshot` to build all release bundles locally.
