# Copilot instructions for `bbl`

## Project context
- Go monorepo for an institutional repository / academic bibliography (CRIS-adjacent).
- Core entities: `Work`, `Person`, `Project`, `Organization`, `User`.
- UGent-specific code lives in `ugent/`; everything else is generic platform code.
- Under active development; prefer evolvable designs over premature hardening.
- Prefer small, focused changes that fit existing architecture and naming.

## Architecture
- Domain types and logic live in the root `bbl` package. No ORM — plain SQL via pgx.
- `app/` is the web layer (templ + htmx). Server-rendered; no SPA patterns.
- `cmd/bbl/cli/` has the CLI (Cobra). Config is YAML-based, loaded via `BBL_CONFIG` env var.
- Background tasks use catbird. Real-time uses centrifugo.
- Search uses OpenSearch via `bbl.Index` interface (`opensearchindex/` implementation).
- Migrations use Goose (SQL + Go in `migrations/`).

## Assertion model
Every field value is stored as an **assertion row** — not as a column on the entity table.
Full design doc: `docs/assertion-model.md`.

Key concepts:
- Each assertion has a bigint ID (from a shared sequence), a value, and either a `*_source_id` (FK to `*_sources`) or a `user_id` (FK to `bbl_users`). Exactly one is set.
- **Scalar fields** store their value inline in the assertion row (`val jsonb`).
- **Collective fields** (identifiers, contributors, titles, etc.) store values in dedicated relation tables with `assertion_id` FK.
- **Pinning** determines which assertion is displayed. Priority: recent curator > curator > recent user > user > source by priority. The `role` column on the assertion row stores the user's role at assertion time.
- **No `pinned_by` column.** Pin authority is derived: `user_id IS NOT NULL` = human, `*_source_id IS NOT NULL` = source.
- **Additive human assertions**: each human edit creates a new assertion row. Previous assertions stay as history. Only Unset deletes.
- **Copy-on-write**: when a human asserts a value (including selecting an existing source value), they create their own assertion row. For collectives, the entire list is copied.

### Updates
- State changes go through `Update` with typed updater structs.
- Updaters use **concrete named types** per field: `SetWorkVolume`, `UnsetWorkVolume`, `SetWorkTitles`, etc. No generic `SetWorkField` with a field name parameter.
- **Set** creates a new assertion row. **Hide** creates an assertion with `hidden=true`. **Unset** removes the human assertion; auto-pin re-evaluates.
- Required fields (work titles, person name, project titles, org names) have no Unset.
- Wire format: `{"set": "work:volume", "id": "...", "val": "42"}`.

### Import path
- `ImportWorks`/`ImportPeople`/etc. take `iter.Seq2` iterators.
- Import creates source-linked assertions (`*_source_id` set, `user_id = NULL`).
- On re-import, existing source assertions are deleted and replaced; human assertions are untouched.
- Shared write helpers (e.g. `writeWorkTitle`, `writeWorkIdentifier`) are used by both updates and import.

### Key files
- `assertion.go` — auto-pin logic
- `updaters.go` — updater interface, `updateEffect`, types
- `work_field_updaters.go` — scalar field Set/Unset for works
- `work_relation_updaters.go` — collective Set/Unset for works + shared write helpers
- `person_field_updaters.go`, `project_field_updaters.go`, `organization_field_updaters.go` — same pattern
- `import.go` — import helpers, cache rebuild
- `revs.go` — `Update()` method, revision creation
- `update_decoding.go` — JSON wire format decoder
- `docs/assertion-model.md` — full design doc

## Coding principles
- Follow existing Go style in the touched area. No ORM, plain SQL via pgx.
- Touch only files relevant to the request. Avoid unrelated cleanup.
- At architecture decision points, ask instead of locking in irreversible design choices.
- Do not add comments unless needed for non-obvious behavior.
- Do not add dependencies without clear justification.
- Do not add license/copyright headers.
- Return sentinel errors bare (`return nil, ErrNotFound`). Wrap unexpected errors with method name: `fmt.Errorf("MethodName: %w", err)`.

## State change invariants
- All state changes go through `Update` (human) or `Import*` (source). No ad-hoc direct writes.
- One revision = one transaction boundary. Every assertion carries its `rev_id`.
- Auto-pin runs after every write to re-evaluate which assertions are displayed.
- Human assertions are additive — history builds up naturally from assertion rows.

## Data model rules
- Inspect the PostgreSQL schema (`migrations/00001_schema.sql`) before proposing structural changes.
- Work profiles define valid fields per work kind — changes to work metadata must respect profile-driven behavior.
- `bbl_works.cache jsonb` is a denormalized display cache rebuilt from pinned assertions via `bbl_works_view`.
- Person/organization deduplication is critical; preserve provenance and make source/precedence decisions explicit.

## Change scope
- Touch only files relevant to the request.
- Avoid unrelated cleanup in the same change.
- At architecture decision points, ask instead of locking in irreversible design choices.

## Validation
- `go test ./...` for Go changes.
- `npm run build` for frontend asset changes.
- Mention failing tests that are unrelated to the change.

## Developer workflow
- `make dev` — runs server + asset watcher with hot reload.
- `go run ./ugent/cmd/bbl migrate up` — run migrations.
- `go run ./ugent/cmd/bbl seed` — load seed data.
- `docker compose up -d` — start local services.

## Migrations
- Keep migrations explicit and reversible.
- Do not modify secrets, env files, or sample data unless asked.
- Do not run destructive commands against external services.
