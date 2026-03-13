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

Key concepts:
- Each assertion has a UUID, a value, and either a `*_source_id` (FK to `*_sources`) or a `user_id` (FK to `bbl_users`). Exactly one is set.
- **Scalar fields** (volume, year, etc.) go in `bbl_*_fields` tables with `(entity_id, field, val jsonb)`.
- **Collective fields** (identifiers, contributors, titles, etc.) go in dedicated tables (e.g. `bbl_work_identifiers`, `bbl_work_contributors`).
- **Pinning** determines which assertion is displayed. Human assertions always win; otherwise the highest-priority source wins. Auto-pin runs after every write.
- **No `pinned_by` column.** Pin authority is derived: `user_id IS NOT NULL` = human (always wins), `*_source_id IS NOT NULL` = source (priority-based).

### Mutations
- State changes go through `AddRev` with typed mutation structs.
- Mutations use **concrete named types** per field: `SetWorkVolume`, `DeleteWorkVolume`, `SetWorkTitles`, etc. No generic `SetWorkField` with a field name parameter.
- **Set** always expects a value. **Delete** removes the human assertion; auto-pin re-evaluates.
- Required fields (work titles, person name, project titles, org names) have no Delete mutation.
- Both human (AddRev) and import paths write mutation records to `bbl_mutations` for replayable history.

### Import path
- `ImportWorks`/`ImportPeople`/etc. take `iter.Seq2` iterators.
- Import creates source-linked assertions (`*_source_id` set, `user_id = NULL`).
- On re-import, existing source assertions are deleted and replaced; human assertions are untouched.
- Shared write helpers (e.g. `writeWorkTitle`, `writeWorkIdentifier`) are used by both mutations and import.

### Key files
- `assertion.go` — assertion types
- `mutations.go` — mutation interface, `mutationEffect`, `AddRev`
- `work_field_mutations.go` — scalar field Set/Delete for works
- `work_relation_mutations.go` — collective Set/Delete for works + shared write helpers
- `person_field_mutations.go`, `project_field_mutations.go`, `organization_field_mutations.go` — same pattern
- `import.go` — auto-pin helpers, cache rebuild
- `docs/assertion-model.md` — full design doc

## Coding principles
- Follow existing Go style in the touched area.
- Respect command-query separation.
- Fix root causes, not surface workarounds.
- Do not add comments unless needed for non-obvious behavior.
- Do not add dependencies without clear justification.
- Do not add license/copyright headers.

## State change invariants
- All state changes go through `AddRev` (human) or `Import*` (source). No ad-hoc direct mutations.
- One revision = one transaction boundary.
- All mutations are recorded in `bbl_mutations` for audit and replay.
- Auto-pin runs after every write to re-evaluate which assertions are displayed.

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
