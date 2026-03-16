# TODO

## Assertion model (remaining)

- [ ] **Decision**: Import pipeline mutation granularity
  - **A) Coarse records (current)**: keep single `ImportWork`/`ImportPerson`/etc. mutation records with `diff: '{}'`. Replay by re-running import from the stored source record in `*_sources`. Simpler, but pinning history is implicit.
  - **B) Per-field records**: write `SetWorkVolume`, `SetWorkTitles`, etc. during import, same as UI mutations. Explicit field-level audit trail and pinning history, but more rows and import is slower.
- [ ] Auto-pin integration tests (human wins over source, re-import re-evaluates, etc.)
- [ ] Rights check: curator assertion can only be replaced by curator
- [ ] Add context to assertion deletion: is it a delete or an assertion the field does not exist?

## Infrastructure

- [ ] Mock ugent_ldap source
- [ ] Mock plato source

## Other

- [ ] Candidates
- [ ] Authorization layer

## Features to port from prototype

### Core backend

- [ ] `rev add` CLI — apply revisions from JSONL stdin
- [ ] Background tasks via Catbird (reindex, import, export, notifications, generate representations)

### Backoffice UI

- [ ] Work batch edit
- [ ] Work CSL export
- [ ] Work change history/audit view
- [ ] Contributor CRUD with person suggest/autocomplete
- [ ] File upload (S3 presigned URLs) + attach/detach
- [ ] User curated lists (CRUD, export, add items)
- [ ] Work kind change
- [ ] Impersonation

### External protocols & APIs

- [ ] OAI-PMH server (full verb set + resumption tokens)
- [ ] SRU server (CQL queries)
- [ ] ORCID API client
- [ ] Webhook subscriptions + async delivery

### Infrastructure

- [ ] Form binding library (`bind/`)
