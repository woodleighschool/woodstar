# Woodstar ⭐️

Self-hosted macOS management for the gaps Intune leaves: Munki for software and patching, Santa for what's allowed to run, and osquery for inventory. Built at Woodleigh to fill the holes after moving Mac management from Jamf to Intune.

## Quick start

```bash
brew install mise
mise trust
mise install
mise run deps
```

Run against local Postgres:

```bash
docker compose up -d postgres

WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_PUBLIC_URL='http://localhost:8080' \
WOODSTAR_SESSION_SECRET='replace-with-at-least-32-characters' \
  mise exec -- go run ./cmd/woodstar serve
```

## Documentation

The full documentation is a Docusaurus site under [`docs/`](docs/): how it fits together, the agent protocols, configuration, AutoPkg, and a generated API reference. Start with [`docs/content/intro.md`](docs/content/intro.md).
