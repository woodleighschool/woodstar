# woodstar ⭐️

[![Release](https://img.shields.io/github/v/release/woodleighschool/woodstar?display_name=tag&sort=semver)](https://github.com/woodleighschool/woodstar/releases/latest)
[![CI](https://github.com/woodleighschool/woodstar/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/woodleighschool/woodstar/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/woodleighschool/woodstar?logo=go)](https://github.com/woodleighschool/woodstar/blob/main/go.mod)
[![Container](https://img.shields.io/badge/container-ghcr.io-2496ED?logo=github&logoColor=white)](https://github.com/orgs/woodleighschool/packages/container/package/woodstar)
[![License](https://img.shields.io/github/license/woodleighschool/woodstar)](https://github.com/woodleighschool/woodstar/blob/main/LICENSE)

Manages macOS devices with Munki, Santa, and Orbit/osquery.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## 🌱 What's inside

- Host enrollment, hardware, software, users, and query results
- Munki repositories, manifests, packages, and assignments
- Santa rules, client configuration, sync, and events
- osquery reports, policies, live queries, and dynamic labels
- Entra directory sync for people, groups, and user affinity
- Distribution-point workers for local package delivery

## 🚀 Usage

Download server archives for macOS or Linux from the [latest release](https://github.com/woodleighschool/woodstar/releases/latest), or use the container `ghcr.io/woodleighschool/woodstar:rolling`.

### Docker Compose

Start from the example environment:

```bash
cp .env.example .env
openssl rand -hex 32
```

Before starting, set these values in `.env`:

| Variable                          | Value                                              |
| --------------------------------- | -------------------------------------------------- |
| `WOODSTAR_URL`                    | HTTPS address used by browsers and managed clients |
| `WOODSTAR_TLS_CERT_FILE`          | Certificate for that address                       |
| `WOODSTAR_TLS_KEY_FILE`           | Matching private key                               |
| `WOODSTAR_STORAGE_CAPABILITY_KEY` | Output from `openssl rand -hex 32`                 |

```bash
docker compose up -d
docker compose exec woodstar /woodstar user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

The [getting started guide](https://woodleighschool.github.io/woodstar/docs/getting-started/docker-compose) covers certificates, first sign-in, and the optional distribution point worker.

## ⚙️ Configuration

The [documentation](https://woodleighschool.github.io/woodstar/) covers configuration, storage, client protocols, AutoPkg, and the API.

## 🧑‍💻 Development

Mise owns the toolchain and commands:

```bash
mise install
mise run dev
mise run test
mise run lint
```

Backend code lives under `cmd/woodstar` and `internal/`; the React app lives under `web/`; the documentation site lives under `docs/`.

See the [development setup](https://woodleighschool.github.io/woodstar/docs/development/setup) and [command reference](https://woodleighschool.github.io/woodstar/docs/development/commands) for PostgreSQL, end-to-end, storage, generation, and docs workflows.

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).

## 🙏 Acknowledgments

- **[Munki](https://github.com/munki/munki)** - Managed software installation for macOS
- **[MunkiAdmin](https://github.com/hjuutilainen/munkiadmin)** - Graphical editor for Munki repositories
- **[AutoPkg](https://github.com/autopkg/autopkg)** - Automation framework for macOS software packaging
- **[Santa](https://github.com/northpolesec/santa)** - Binary authorization and endpoint security for macOS
- **[osquery](https://github.com/osquery/osquery)** - SQL-powered operating system instrumentation
- **[Fleet](https://github.com/fleetdm/fleet)** - Open-source device management platform and home of Orbit
- **[Zentral](https://github.com/zentralopensource/zentral)** - Event-driven platform for endpoint management
