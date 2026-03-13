# TODO

## Assertion model (remaining)

- [ ] Import pipeline: write mutation records to `bbl_mutations` (replayable history)
- [ ] Auto-pin integration tests (human wins over source, re-import re-evaluates, etc.)
- [ ] Rights check: curator assertion can only be replaced by curator

## Features to port from prototype

## Core backend

- [ ] Full search CLI commands (works, people, projects, organizations, users)
- [ ] `rev add` CLI — apply revisions from JSONL stdin
- [ ] Single-record CLI commands (`person`, `project`, `organization`, `user` by ID)
- [ ] Reindex CLI commands (per entity type)
- [ ] Background tasks via Catbird (reindex, import, export, notifications)
- [ ] Authorization layer (`can.Curate`, `can.ViewWork`, `can.EditWork`)

## Backoffice UI

- [ ] People, projects, organizations, users management pages
- [ ] Work batch edit
- [ ] Work export (CSV, CSL, OAI-DC)
- [ ] Work change history/audit view
- [ ] Contributor CRUD with person suggest/autocomplete
- [ ] File upload (S3 presigned URLs) + attach/detach
- [ ] User curated lists (CRUD, export, add items)
- [ ] Work kind change

## Public discovery

- [ ] Public works list/search with access control
- [ ] Public work detail page

## External protocols & APIs

- [ ] OAI-PMH server (full verb set + resumption tokens)
- [ ] SRU server (CQL queries)
- [ ] ORCID API client
- [ ] ArXiv importer

## Export / encoding

- [ ] CSL encoder (external citeproc service)
- [ ] OAI-DC encoder
- [ ] CSV exporter
- [ ] Representations storage (generated metadata formats)

## Infrastructure

- [ ] Webhook subscriptions + async delivery
- [ ] Zero-downtime reindex (alias switching)
- [ ] Form binding library (`bind/`)
- [ ] i18n system
- [ ] Pagination helper
- [ ] Session/auth (OIDC, cookies, admin impersonation)

## Other

- [ ] Candidates (work + person)
