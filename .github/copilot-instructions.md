# Copilot instructions for `bbl`

## Project context
- This repository is a Go monorepo for the bbl platform and related modules (`app`, `biblio`, `orcid`, `opensearchindex`, etc.).
- The application is an institutional repository and academic bibliography with scope overlap with CRIS systems.
- It is the authoritative source for institutional research output.
- Core entities are `Work` (publication or dataset), `Person`, `Project`, `Organization`, and `User`.
- The project is a prototype under active development; prefer evolvable designs and explicit trade-offs over premature hardening.
- Prefer small, focused changes that fit existing architecture and naming.
- Refactors are allowed when they clearly improve maintainability and stay scoped to the task.

## Architecture overview
- Keep the split between generic platform code and UGent-specific integrations clear (for example: generic modules in root packages, institution-specific behavior in `biblio/`).
- The platform combines multiple concerns (authorities, backoffice, discovery, OAI, APIs) in one application; prefer cohesive integration over scattered standalone implementations.
- Treat `app/` as the web frontend/UI layer and Go packages as the domain/backend layers.
- There are two web app contexts: a public frontend (currently mostly stub) and a backoffice application.
- Prefer extending existing package boundaries over introducing new cross-cutting abstractions.
- Prefer server-rendered interactions in `app/` using templ + htmx patterns already present in the codebase.

## Tooling defaults
- Use catbird for background tasks, flows, long-running/async processing, and internal messaging.
- Use templ for application templating in `app/views/`.
- Use htmx for server-side partial rendering and interaction patterns.
- Use centrifugo for real-time user interactions.
- Do not introduce client-heavy SPA patterns or frontend state frameworks unless explicitly requested.
- Prefer progressive enhancement for UI interactions: keep core flows server-rendered and functional without custom client-side state where practical.

## Coding principles
- Follow existing Go style and package organization in the touched area.
- Keep public APIs and file structure stable unless a task explicitly asks to change them.
- Respect command-query separation: keep query/read concerns and state-changing command concerns clearly separated.
- Fix root causes, not surface workarounds.
- Avoid adding new dependencies unless there is a clear, justified need.
- Do not add license/copyright headers.
- Do not add comments unless needed for non-obvious behavior.

## State change and audit invariants
- Apply state changes through the established revision flow (for example via `AddRev`) rather than ad-hoc direct mutations.
- Treat one revision as one transaction boundary.
- Preserve and extend auditing guarantees: state changes must remain traceable in the `changes` table.
- When implementing new write paths, align them with existing revision/change-history patterns before introducing new mechanisms.

## Change scope
- Touch only files relevant to the request.
- Avoid unrelated cleanup in the same change.
- Preserve existing behavior for unaffected features.
- If requirements are ambiguous, implement the simplest viable option and call out assumptions.
- At architecture decision points, ask the user to choose between options instead of locking in irreversible design choices.

## Data-centered design defaults
- Treat this as a data-centered application: inspect and align with the PostgreSQL schema before proposing structural changes.
- Infer business rules from existing schema constraints, identifiers, and revision/audit tables; do not bypass them with application-only shortcuts.
- When schema and code appear to diverge, prefer preserving persisted data invariants and ask for clarification before changing semantics.

## Identity and authority focus
- Person and organization deduplication is a critical problem area; prioritize robustness and traceability over convenience.
- Assume imported person data can contain duplicates, errors, and conflicting identifiers across sources.
- Current baseline behavior: a single `User` may be linked to multiple `Person` records when identifiers overlap.
- For new identity-linking logic, preserve provenance and make source/precedence decisions explicit.
- Expect iterative design: when requested, provide architecture/design sketches for stronger authority and precedence rules.

## Current constraints to preserve
- Permissions are currently simple and partly implicit (role-based field plus hardcoded checks); avoid introducing hidden permission behavior.
- Search/indexing is currently eventually consistent; document any consistency assumptions and failure modes in changes touching indexing.
- Preserve index-switching assumptions in indexing-related changes and avoid introducing cutover steps that require downtime.
- Work profiles define active fields by work type (`book`, `journal_article`, etc.); changes to work metadata must respect profile-driven behavior.
- Query/list endpoints should prefer cursor/search-after style pagination and avoid deep-offset paging patterns.

## Critical areas to inspect before edits
- Entry points and orchestration: `cmd/bbl/`, `app/app.go`, `biblio/main.go`.
- Entity and repository/domain logic: root-level `*.go` files (`work.go`, `person.go`, `project.go`, `organization.go`, `repo.go`, `query.go`, etc.).
- Integrations and indexers: `orcid/`, `oaipmh/`, `opensearchindex/`, `s3store/`, `ldap/`.
- UI and assets: `app/views/`, `app/assets/`, `app/static/`, `app/*_handlers.go`.

## Validation expectations
- For Go changes, run targeted tests for the touched package(s) by default.
- Typical commands:
  - `go test ./orcid/...` (or the specific package being changed)
  - `go test ./...` only when changes are cross-cutting or when explicitly requested
- If frontend code in `app/` is changed, run:
  - `cd app && npm run build` (and relevant frontend test/lint scripts when available)
- Mention any failing tests that are unrelated to the change instead of silently ignoring them.

## Developer workflow defaults
- Prefer commands already documented in `DEVELOPMENT.md` and `Makefile`.
- For setup- or workflow-related changes, keep command examples aligned with:
  - `go mod tidy`
  - `docker compose up --remove-orphans`
  - `go run cmd/bbl/main.go migrate up`
  - `make live`
- Do not introduce parallel alternative workflows unless explicitly requested.

## Data, migrations, and operations
- Be careful with migration-related changes; keep them explicit and reversible where possible.
- Do not modify secrets, local env files, or sample data unless explicitly asked.
- Do not run destructive commands against external services.

## Documentation sync requirements
- Always update docs when behavior, commands, configuration, or workflows change.
- Keep docs concise and aligned with existing wording in `README.md`, `DEVELOPMENT.md`, and `DOCUMENTATION.md`.
- If a change affects CLI/admin/import-export workflows, include doc updates in the same change.

## Output expectations for generated patches
- Provide minimal, reviewable diffs.
- Reference changed files clearly.
- Include short verification notes (what was run and what passed/failed).

## Common implementation defaults
- Prefer existing helper packages (for example `httperr/`, `pagination/`, `bind/`) over creating duplicate utility logic.
- Keep handler changes consistent with surrounding handler patterns in `app/*_handlers.go`.
- Preserve existing serialization/import/export patterns (`work_encoder`, `work_importer`, `work_exporter`) when extending entity behavior.
- For file handling flows, preserve direct-to-S3 upload/download patterns rather than routing large payloads through the app server.
- For import/export and batch operations, preserve high-volume behavior and avoid introducing unnecessary size limits.
