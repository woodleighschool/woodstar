# woodstar ⭐️

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

Start with the example environment and a storage capability key:

```bash
cp .env.example .env
openssl rand -hex 32
docker compose up -d
docker compose exec woodstar /woodstar user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

Compose uses the published `rolling` image. The [Docker Compose guide](https://woodleighschool.github.io/woodstar/docs/getting-started/docker-compose) covers certificates, hostnames, storage, and first sign-in.

## ⚙️ Configuration

The [documentation](https://woodleighschool.github.io/woodstar/) covers configuration, client protocols, AutoPkg, and the API.

Use the [environment reference](https://woodleighschool.github.io/woodstar/docs/configuration/environment) and [storage guide](https://woodleighschool.github.io/woodstar/docs/configuration/storage) for deployment settings.

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
