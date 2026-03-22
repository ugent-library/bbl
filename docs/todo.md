# TODO


## Model

- [ ] Get rid of field catalog (dynamic fields)
- [ ] Union pinning: resolveUnionPin + field catalog declaring union fields (identifiers, classifications); Update path queueAutoPinForField needs Go-side strategy when union is added. Note: doesn't necessarily require row-per-item — could use per-item override rows on top of a base array (source provides the full list, human overrides cherry-pick individual items).
- [ ] Auto-pin integration tests (human > source, exclusive + union collections)
- [ ] Review/lock mechanism: explicit curator endorsement (separate from assertion)
- [ ] Candidates
- [ ] Investigate: should source re-imports also log to bbl_history? Currently only human edits are tracked there; source history relies on the source record's original payload.
- [ ] Reduce accidental complexity in collection handling. The assertion model is simple (field → assertion rows → pin best one) but the implementation added unnecessary layers:
  - Collections are split into N assertion rows when one row with a JSON array val would work. This created the ordering problem (no explicit order, reconstructed via `ORDER BY a.id` across 3 separate read paths) and order-sensitive noop detection.
  - Three independent read paths reconstruct the same data differently (fetchState, cache rebuild SQL, parseWorkCache). Should converge to one.
  - FK-bearing types split val JSON from extension table FKs then stitch back via enrichVal — a round-trip that exists only because of the row-per-item choice.
  - Consider: one assertion row per collection field (val = full JSON array), FKs as a parallel JSON array or inline. Eliminates ordering, multi-row reconstruction, and the marshal/enrichVal dance.
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
