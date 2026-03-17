# TODO

## Assertion model (remaining)

- [ ] Auto-pin integration tests (human wins over source, re-import re-evaluates, etc.)
- [ ] Rights check: curator assertion can only be replaced by curator
- [ ] Assertion deletion misses context: is it a delete or an assertion the field does not exist?

## Infrastructure

- [ ] Mock ugent_ldap source
- [ ] Mock plato source

## Other

- [ ] Candidates
- [ ] Authorization layer
- [ ] Collections (two types: query-based / dynamic, and manual / rules-based)

## Backoffice UI

## External protocols & APIs

- [ ] OAI-PMH: representation cache table (avoid re-harvest when entity timestamp bumps but encoded output is identical)
- [ ] OAI-PMH: deleted record tracking (currently `DeletedRecord: "no"`; need to surface deleted/privatized works as `<header status="deleted">` so harvesters can clean up)
- [ ] OAI-PMH: sets via collections
- [ ] OAI-PMH: `Identify` description element (oai-identifier, friends)
- [ ] OAI-PMH: HTTP compression support

## Infrastructure

- [ ] Split off sru library
- [ ] Split off oaipmh library

## Features to port from prototype

### Core backend

- [ ] `rev add` CLI — apply revisions from JSONL stdin
- [ ] Background tasks via Catbird (reindex, import, export, notifications, generate representations)

### Backoffice UI

- [ ] Work batch edit
- [ ] Work change history/audit view
- [ ] File upload (S3 presigned URLs) + attach/detach
- [ ] User curated lists (CRUD, export, add items)
- [ ] Work kind change
- [ ] Impersonation

### External protocols & APIs

- [ ] ORCID API client
- [ ] Webhook subscriptions + async delivery

### Infrastructure

- [ ] form binding library (`bind/`)
