# Woodstar ⭐️

Woodstar is a self-hosted macOS observability and admin surface built around
Orbit/osquery first, with Santa and Munki (server) as first class modules.

## Development

```bash
brew install mise
mise trust
mise install
mise run deps
mise run test
mise run backend
```

Run locally with Postgres:

```bash
docker compose up -d postgres

WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
WOODSTAR_SESSION_SECRET='replace-with-at-least-32-characters' \
  mise exec -- go run ./cmd/woodstar serve
```
