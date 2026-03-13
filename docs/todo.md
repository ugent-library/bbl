# TODO

## Assertion model (remaining)

- [ ] Import pipeline: write mutation records to `bbl_mutations` (replayable history)
- [ ] Auto-pin integration tests (human wins over source, re-import re-evaluates, etc.)
- [ ] Rights check: curator assertion can only be replaced by curator

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

### Export / encoding

- [ ] CSL encoder (external citeproc service)

### Infrastructure

- [ ] Form binding library (`bind/`)
- [ ] i18n system
