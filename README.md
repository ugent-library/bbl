# bbl

Institutional repository system for Ghent University Library.

## Prerequisites

- Go 1.25+
- Node.js
- Docker & Docker Compose

## Quick start

```sh
# 1. Start local services (Postgres, OpenSearch, mock OIDC, S3, etc.)
docker compose up -d

# 2. Setup config
cp ugent/config.yaml.example ugent/config.yaml
export BBL_CONFIG="ugent/config.yaml"
export UGENT_LDAP_URL="****"
export UGENT_LDAP_USERNAME="****"
export UGENT_LDAP_PASSWORD="****"
export PLATO_URL="****"
export PLATO_USERNAME="****"
export PLATO_PASSWORD="****"

# 3. Install dependencies
go mod download
npm install

# 4. Build assets and generate templ files
make build

# 5. Run database migrations and load seed data
go run ./ugent/cmd/bbl migrate up
go run ./ugent/cmd/bbl seed

# 6. Run (auto-reloads on file changes)
make dev
```

This starts the Go server (with templ generation) and the esbuild watcher at http://localhost:3000.
Log in via the mock OIDC provider at http://localhost:3000/backoffice (enter admin or researcher).

## Local services

| Service | Port | Purpose |
|---|---|---|
| PostgreSQL | 3351 | Primary database (`bbl/bbl/bbl`) |
| OpenSearch | 3352 | Search index (2-node cluster) |
| Mock OIDC | 3350 | Authentication (interactive login) |
| LocalStack S3 | 3371 | File storage |
| Centrifugo | 3357 | Real-time (WebSocket) |
| citeproc | 8085 | Citation formatting |

## Project structure

```
cmd/bbl/cli/       CLI commands (Cobra)
ugent/             UGent-specific binary, config, sources
  cmd/bbl/         Custom entrypoint registering UGent sources
  config.yaml      UGent config (see config.yaml.example)
  profiles.yaml    Work kind profiles
  plato/           Plato API work source
app/               HTTP handlers + views (templ)
  views/           Templ templates
  assets/          JS/CSS source (esbuild)
  static/          Built assets (generated)
migrations/        SQL + Go migrations (Goose)
opensearchindex/   OpenSearch index implementation
ldap/              LDAP user source
oidcauth/          OIDC auth provider
docs/              Design docs and TODOs
```

## Key commands

```sh
make dev              # Run dev server + asset watcher
make build            # Build everything (assets + templ + go)

# CLI (after building)
bbl start             # Start the server
bbl migrate up        # Run migrations
bbl migrate down      # Rollback migrations
bbl seed              # Seed test data
bbl works import SRC  # Import works from stdin JSONL
bbl reindex works     # Reindex works in OpenSearch
```

## Configuration

The app is configured via YAML. The config path is set via `BBL_CONFIG` env var.
Copy `ugent/config.yaml.example` to `ugent/config.yaml` and fill in the credentials.

Key config sections:
- `conn` — Postgres connection string
- `profiles` — path to work kind profile definitions
- `opensearch` — OpenSearch addresses
- `user_sources` — LDAP or other user sources
- `work_sources` — Plato or other work sources
- `auth` — OIDC providers

## Tests

```sh
go test ./...
```
