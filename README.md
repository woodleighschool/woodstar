# Woodstar ⭐️

Self-hosted macOS management for the gaps Intune leaves. Woodstar brings together Munki for software, Santa for execution policy, and Orbit with osquery for enrollment, inventory, and checks.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## 🌱 What's inside

- **Hosts and inventory** from Orbit or osquery, with hardware, software, users, and query results in one place.
- **Munki** manifests, catalogs, packages, icons, client resources, and software assignments.
- **Santa** rules, client configuration, sync, and execution events.
- **osquery** reports, scheduled checks, live queries, and dynamic labels.
- **Labels** shared across software, rules, reports, and checks.
- **Entra directory sync** for people, groups, user affinity, and label membership.

## 🏡 Running Locally

Start with the example environment and a storage capability key:

```bash
cp .env.example .env
openssl rand -hex 32
```

Add the generated key to `WOODSTAR_STORAGE_CAPABILITY_KEY`, check the URL and certificate paths, then start Woodstar and PostgreSQL:

```bash
docker compose up -d
docker compose exec woodstar /woodstar user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

Compose uses the published `rolling` image. Uncomment the build block in [`docker-compose.yml`](docker-compose.yml) to build the current checkout instead.

The [Docker Compose guide](https://woodleighschool.github.io/woodstar/docs/getting-started/docker-compose) covers certificates, hostnames, storage, and first sign-in.

## 📚 Documentation

The [Woodstar docs](https://woodleighschool.github.io/woodstar/) cover the web app, configuration, client protocols, AutoPkg, and the API. For code changes, head to the [development setup](https://woodleighschool.github.io/woodstar/docs/development/setup) and [command reference](https://woodleighschool.github.io/woodstar/docs/development/commands).

## 🧑‍💻 Development

[mise](https://mise.jdx.dev/) owns the toolchain and everyday commands:

```bash
mise install
mise run dev
mise run test
mise run lint
```

Backend code lives under `cmd/woodstar` and `internal`; the React app lives under `web`. The full command list includes the PostgreSQL, end-to-end, and storage integration lanes.

## 🤝 Contributing

Contributions are welcome! Please open an [issue](https://github.com/woodleighschool/woodstar/issues) before starting a larger change.

## 📄 License

Woodstar is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **[Munki](https://github.com/munki/munki)** - Managed software installation for macOS
- **[MunkiAdmin](https://github.com/hjuutilainen/munkiadmin)** - Graphical editor for Munki repositories
- **[AutoPkg](https://github.com/autopkg/autopkg)** - Automation framework for macOS software packaging
- **[Santa](https://github.com/northpolesec/santa)** - Binary authorization and endpoint security for macOS
- **[osquery](https://github.com/osquery/osquery)** - SQL-powered operating system instrumentation
- **[Fleet](https://github.com/fleetdm/fleet)** - Open-source device management platform and home of Orbit
- **[Zentral](https://github.com/zentralopensource/zentral)** - Event-driven platform for endpoint management

---
