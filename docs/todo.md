# TODO

## Model

- [ ] Update rules: diff against pinned, only assert differences (see `docs/update-rules.md`)
- [ ] `needs()` returns `*updateNeeds` with entity IDs; `Update()` batch-fetches full entities
- [ ] `matches()` on each updater to detect noops against pinned values (scalars + collectives)
- [ ] Add missing `Hide*` updaters for all entity types
- [ ] Auto-pin integration tests (role-aware: curator > user > source, recency within same role)
- [ ] Wire `role` through `Update()` — pass the user's role at assertion time (currently nil)
- [ ] Assertion history GC: prune unpinned rows for same (entity, field, user, role)
- [ ] Review/lock mechanism: explicit curator endorsement (separate from assertion)
- [ ] Candidates
- [ ] Authorization layer
- [ ] Collections (two types: query-based / dynamic, and manual / rules-based)

## Backoffice UI

- [ ] Work batch edit: scalar CSV (CLI: `bbl works batch-export` / `bbl works batch-import`)
- [ ] Work batch edit: collective fields (separate CSV per type: titles, keywords, contributors, etc.)
- [ ] Work batch edit: web UI (download/upload in backoffice)
- [ ] Work change history/audit view (repo method + templ page, link from detail page)
- [ ] Form edit: render curator-pinned fields as read-only for non-curator users
- [ ] File upload (S3 presigned URLs) + attach/detach
- [ ] User curated lists (CRUD, export, add items)
- [ ] Work kind change
- [ ] Impersonation

## External protocols & APIs

- [ ] OAI-PMH: representation cache table (avoid re-harvest when entity timestamp bumps but encoded output is identical)
- [ ] OAI-PMH: deleted record tracking (currently `DeletedRecord: "no"`; need to surface deleted/privatized works as `<header status="deleted">` so harvesters can clean up)
- [ ] OAI-PMH: sets via collections
- [ ] OAI-PMH: `Identify` description element (oai-identifier, friends)
- [ ] OAI-PMH: HTTP compression support
- [ ] ORCID API client
- [ ] Webhook subscriptions + async delivery

## Infrastructure

- [ ] Split off sru library
- [ ] Split off oaipmh library
- [ ] Mock ugent_ldap source
- [ ] Mock plato source
