# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Dev (hot reload server + asset watcher)
make dev

# Build everything (assets + templ + go binary)
make build

# Tests
go test ./...

# Single test
go test -run TestFoo ./path/to/package

# Migrations
go run ./ugent/cmd/bbl migrate up
go run ./ugent/cmd/bbl migrate down

# Seed data
go run ./ugent/cmd/bbl seed

# Start local services
docker compose up -d

# Frontend assets only
npm run build
```

Config path is set via `BBL_CONFIG` env var (e.g. `ugent/config.yaml`).

## Architecture

Go monorepo for an institutional repository (CRIS-adjacent). Server-rendered web app (templ + htmx), no SPA. PostgreSQL via pgx (no ORM), OpenSearch for search, Goose for migrations.

**Core entities**: Work, Person, Project, Organization, User.

### Package layout

- Root `bbl` package: domain types, data access (SQL), import/export logic
- `app/`: HTTP handlers + templ views, session/auth
- `cmd/bbl/cli/`: Cobra CLI, YAML config loading, service wiring via `Registry`
- `opensearchindex/`: OpenSearch `bbl.Index` implementation
- `ldapsource/`: LDAP user source
- `oidcauth/`: OIDC auth provider
- `migrations/`: Goose SQL + Go migrations
- `ugent/`: UGent-specific code (custom binary, Plato work source, config, profiles)

### Assertion model

Field values are stored as **assertion rows**, not columns on entity tables. This is the central design pattern. Full design doc: `docs/assertion-model.md`.

- **Asserters**: either a source (automated feed, FK to `bbl_*_sources`) or a human (FK to `bbl_users`). `CHECK (num_nonnulls(*_source_id, user_id) = 1)` enforces exactly one.
- **Scalar fields** go in `bbl_*_fields` tables `(entity_id, field, val jsonb)`. Grouping key: `(entity_id, field)`.
- **Collective fields** (identifiers, contributors, titles, abstracts, notes, keywords, classifications, FK relations) go in dedicated relation tables. Grouping key: `(entity_id)` per table — one asserter's entire list wins.
- **Pinning** determines display value. Always implicit (side effect of writes, never explicit). Auto-pin rule: human assertion exists → it wins; otherwise highest-priority source (`bbl_sources.priority`) wins.
- **Copy-on-write**: when a human asserts, they create their own assertion row. For collectives, the entire list is copied. Pin is always on the human's row, never on a source's.
- **Replace semantics**: one human assertion slot per grouping key. New human assertion = DELETE old + INSERT new.
- **Re-import**: DELETE all of this source record's assertions + INSERT new ones. Human assertions are untouched.
- **Delete**: remove the assertion row → auto-pin re-evaluates → next best assertion gets pinned (or field absent).
- **`status`/`review_status`** are state columns, NOT assertions — they don't participate in pinning. They have their own mutations (`SetWorkStatus`, `SetWorkReviewStatus`).
- **`kind`** is a regular assertion in `bbl_*_fields`, but on entity creation the system creates its own copy-on-write kind assertion to prevent silent kind changes on re-import.
- `cache jsonb` on entity table holds pinned values, rebuilt on every write.

**Mutations**: concrete named types per field (`SetWorkVolume`, `DeleteWorkVolume`), not generic field-name params. Required fields (work titles, person name, project titles, org names) have no Delete mutation. Both human (`Mutate`) and import paths write mutation records to `bbl_mutations` for audit/replay.

Key files: `assertion.go`, `mutations.go`, `*_field_mutations.go`, `*_relation_mutations.go`, `import.go`.

### ID type

`type ID [16]byte` in `id.go`. Generated as ULID, stored as PostgreSQL `uuid`, string representation is Crockford base32.

### Write paths

All state changes go through `Mutate` (human) or `Import*` (source). No ad-hoc direct mutations. One revision = one transaction boundary.

- **Human path (`Mutate`)**: assertion rows get `user_id` set, `*_source_id = NULL`. Replace semantics.
- **Import path**: assertion rows get `*_source_id` set, `user_id = NULL`. Re-import deletes all assertions for the source record and inserts new ones. Shared low-level write helpers used by both paths.
- **Ingestion flow**: incoming record → evaluate → auto-accept (match + import), auto-reject, or candidate (ambiguous, staged for review).

### CLI / Registry pattern

`cmd/bbl/cli/` uses a `Registry` (exported) passed to `NewRootCmd`. Sources are registered via `RegisterUserSource[C]` / `RegisterWorkSource[C]` generic functions. `ugent/cmd/bbl/main.go` registers UGent-specific sources and calls `NewRootCmd`.

### Frontend approach

Server-rendered HTML (templ + htmx), no SPA. Follows the hypermedia-driven approach from https://hypermedia.systems/.

- **htmx** for hypermedia interactions: navigation, form submissions, partial page updates where the server is the source of truth for application state.
- **Plain JS** for widget-level behavior: autocomplete, drag-and-drop, client-side validation.
- At the widget level it's a pragmatic choice — use htmx when it's simpler, JS when it's simpler. No dogma.
- JS is organized using event delegation on `document` (see `app/assets/js/app.js`). Data attributes drive behavior, no framework.
- Assets are built with esbuild (`npm run build`). Dev mode hot-reloads from disk.

## Coding conventions

- Follow existing Go style in the touched area. No ORM, plain SQL via pgx.
- Touch only files relevant to the request. Avoid unrelated cleanup.
- At architecture decision points, ask instead of locking in irreversible design choices.
- Do not add comments unless needed for non-obvious behavior.
- Do not add dependencies without clear justification.
- Do not add license/copyright headers.
- Inspect `migrations/00001_schema.sql` before proposing structural changes.
- Work profiles define valid fields per work kind — respect profile-driven behavior.
- Return sentinel errors bare (`return nil, ErrNotFound`). Wrap unexpected errors with method name: `fmt.Errorf("MethodName: %w", err)`.
- Keep migrations explicit and reversible.
