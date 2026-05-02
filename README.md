# Woodstar ⭐️

Woodstar is a self-hosted macOS observability and admin surface built around
Orbit/osquery first, with Santa and Munki (server) as first class modules.

## Development

```bash
make deps
make test
make backend
```

Run locally with Postgres:

```bash
WOODSTAR_DATABASE_URL='postgres://woodstar:woodstar@localhost:5432/woodstar?sslmode=disable' \
  go run ./cmd/woodstar serve
```
