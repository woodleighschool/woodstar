# stemma-woodstar 🌿

Publish native Munki pkginfo and matching installers through the administrative API.

## 🚀 Usage

See the [Stemma guide](https://woodleighschool.github.io/woodstar/docs/admin/stemma) for configuration, acquisition, and publication.

The executable exposes `woodstar.munki` through Stemma's operation protocol v2.
Select the JSON artifact produced by `munki.pkginfo` and supply the installer as
`inputs.installer`. Native rendering belongs to Stemma; the shared server importer
maps native fields to sparse API mutations. Destination metadata contains only
deployment `targets`.

## 🧑‍💻 Development

Run from this directory:

```sh
mise install
mise run build
mise run test
mise run lint
```

The binary is written to `build/stemma-woodstar`. The module uses the server models
from this checkout and pins the operation SDK in `go.mod`.
Use `mise run test-postgres` for the PostgreSQL integration test.

## 📦 Packaging

GoReleaser builds static `plugin` executables (`plugin.exe` on Windows) and
deterministic tar.zst bundles for Darwin, Linux, and Windows on amd64 and arm64.
Each bundle contains the executable and licence at its root.

ORAS publishes the bundles to `ghcr.io/woodleighschool/woodstar/stemma` using
Stemma’s OCI artifact contract. The release workflow publishes one platform index
under the release tag after every bundle succeeds. Registry authentication uses
the standard credential store; no container runtime is required.

Run `mise run snapshot` to build and inspect all bundles locally without publishing.
