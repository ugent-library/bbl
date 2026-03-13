# Assertion Model

## Rules

### Everything is an assertion

Every field value is an assertion. Assertions coexist side by side. Nothing is
overwritten — each asserter maintains its own assertion independently.

An asserter is either a **source** (automated data feed, identified by a source
record in `bbl_*_sources`) or a **human** (user or curator, identified by
`user_id`). Humans are not sources — `bbl_sources` only contains automated
data sources.

### Every assertion has an ID

Each assertion is a row with a UUID primary key. This gives stable references
for updates, deletes, and pinning. There is no ambiguity about absence vs
deletion vs zero — the assertion either exists or it doesn't.

### Assertion rows track their origin

Each assertion has exactly one of:
- `*_source_id` — FK to the entity's `*_sources` table (source assertion)
- `user_id` — FK to `bbl_users` (human assertion)

A `CHECK (num_nonnulls(*_source_id, user_id) = 1)` enforces this. Source
identity (which source, which priority) is derived by joining through the
source record — no `source` column on assertion tables.

### Pinning selects the display value

One assertion per grouping key is **pinned** — the value used for display,
search, and export. All other assertions remain stored but are not displayed.

Pinning is always implicit — a side effect of writes, never an explicit
operation.

### One kind of pin: the whole field

There is only one pinning granularity: the whole field. No additive, no
keyed, no exceptions.

- **Scalar** (`str_fields`): grouping key = `(entity_id, field)`. One value
  wins.
- **Collective** (identifiers, contributors, titles, abstracts, notes,
  keywords, classifications, FK relations): grouping key = `(entity_id)` per
  table. One asserter's entire list wins.

### Copy-on-write: human assertion = human pin

When a human asserts a value (including selecting an existing source value),
they create their own assertion. The pin is on the human's assertion, never
on a source's.

For collectives, copy-on-write means the human's entire list is copied. A
human touches a list of 3000 authors — it is now a full copy under the
human's user_id. Not storage efficient, but crystal clear.

Consequences:
- Source assertions can only be auto-pinned or not pinned
- Human assertions always win over source assertions
- You can only delete what you asserted — no authority check needed on delete
- Re-import freely replaces source assertions without breaking human pins

### Replace semantics for human assertions

One human assertion slot per grouping key. When a new human asserts, the old
human assertion is replaced (DELETE + INSERT). Rights check at app layer: if
the existing human assertion was made by a curator (look up user's role),
only another curator can replace it.

### Auto-pin rule

Uniform for all field types:

1. Human assertion exists for the grouping key → it is pinned, done
2. No human assertion → highest-priority source's assertion(s) are pinned

Source priority comes from `bbl_sources.priority`, looked up by joining
through `*_sources`.

### Pin authority

No `pinned_by` column. Authority is derived from the assertion row:

- `*_source_id IS NOT NULL` → source assertion → can only be auto-pinned
- `user_id IS NOT NULL` → human assertion → always wins over source

The curator vs user distinction is a rights check (look up the user's role
in the application layer), not stored state on the assertion.

### Delete behavior

Always the same:
1. Delete the assertion row (you can only delete your own)
2. Auto-pin re-evaluates for that grouping key
3. Other assertions exist → highest priority source gets auto-pinned
4. No other assertions → field absent
5. Cache rebuilt

### Re-import

Re-import = delete all of this source record's assertions for the entity +
insert new ones. Source records are identified by `*_sources.id`.

```
DELETE FROM bbl_work_fields WHERE work_source_id = $1
DELETE FROM bbl_work_identifiers WHERE work_source_id = $1
... (all assertion tables — one FK, one value)
INSERT new assertion rows with work_source_id set
```

- Auto-pinned fields: auto-pin re-evaluates
- Human-asserted fields: untouched (pin is on human assertion)

### Contributors

Contributors are a collective — each source asserts an ordered list, not
individual rows. Auto-pin picks the highest-priority source's entire list.
All source lists coexist in storage.

When a human edits contributors, they get a full copy-on-write of the list.
Individual rows within the human copy can be created/updated/deleted, but
pinning always operates on the full list.

### Validation

Two layers:
- **Structural**: every assertion is validated on write (correct type,
  valid scheme, etc.)
- **Completeness**: only pinned values count. Checked when status → public.

## Schema

### Sources

```sql
bbl_sources (
  id          text PRIMARY KEY,
  priority    int NOT NULL DEFAULT 0,   -- used only by auto-pin rule
  description text
)
```

No `controlled_fields`. No field-level priorities. Only automated data
sources — humans are not in this table.

### Source records

Each entity type has a `*_sources` table with a synthetic `id` PK:

```sql
bbl_work_sources (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id      uuid NOT NULL REFERENCES bbl_works(id),
  source       text NOT NULL REFERENCES bbl_sources(id),
  source_id    text NOT NULL,
  candidate_id uuid REFERENCES bbl_work_candidates(id),
  record       bytea NOT NULL,
  fetched_at   timestamptz NOT NULL DEFAULT transaction_timestamp(),
  ingested_at  timestamptz NOT NULL DEFAULT transaction_timestamp(),
  UNIQUE (work_id, source, source_id)
)
```

Same pattern for `bbl_person_sources`, `bbl_project_sources`,
`bbl_organization_sources`.

Assertion rows FK to this table via `work_source_id`, `person_source_id`, etc.

### Entity tables

```sql
bbl_works (
  id              uuid PRIMARY KEY,
  kind            text NOT NULL,          -- denormalized from pinned kind assertion
  status          text NOT NULL DEFAULT 'private',
  review_status   text,
  version         int NOT NULL DEFAULT 0, -- bumped on every cache rebuild
  cache           jsonb,                  -- pinned values (scalars + relations)
  created_at      timestamptz NOT NULL,
  updated_at      timestamptz NOT NULL,
  created_by_id   uuid,
  updated_by_id   uuid
)
```

No `attrs jsonb`. No `provenance jsonb`.

`version` is for optimistic concurrency — cached, bumped on every write.

`created_by_id`/`updated_by_id` are cached from the first/last rev's `user_id`
for single-row entity reads.

`status` and `review_status` are state columns, not sourced fields. They
are not assertions and do not participate in pinning. They have their own
mutations (`SetWorkStatus`, `SetWorkReviewStatus`) that produce audit rows
in `bbl_mutations`.

`kind` is a regular assertion in `bbl_work_fields` — same pinning rules as
any other field. But it is structurally important: the pinned kind determines
the active profile (which fields are valid, which are required).

Special rule: **on entity creation, the system creates its own kind assertion
(copy-on-write from the source) and pins it.** This prevents kind from ever
changing silently on re-import. If a source later asserts a different kind,
it's visible for curation but the entity's effective kind stays stable.
A curator can change kind by creating their own assertion. Kind is cached as
the `kind` column in the entity table.

When pinned kind changes, fields valid under the old profile are not deleted
— they become inactive (still stored, not validated for completeness).

### Scalar assertions

```sql
bbl_work_fields (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  field           text NOT NULL,
  val             jsonb NOT NULL,
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

-- Source: one assertion per source record per field
CREATE UNIQUE INDEX ON bbl_work_fields (work_id, field, work_source_id)
  WHERE work_source_id IS NOT NULL;

-- Human: one assertion per field (replace semantics)
CREATE UNIQUE INDEX ON bbl_work_fields (work_id, field)
  WHERE user_id IS NOT NULL;

-- One pin per field
CREATE UNIQUE INDEX ON bbl_work_fields (work_id, field)
  WHERE pinned = true;
```

Same pattern for `bbl_person_fields`, `bbl_project_fields`,
`bbl_organization_fields`.

### Relation assertions (collective)

Each relation table follows the same `*_source_id` + `user_id` pattern.
Pinning is collective — all rows from the winning asserter are pinned.

```sql
bbl_work_identifiers (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  scheme          text NOT NULL,
  val             text NOT NULL,
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

bbl_work_contributors (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  position        int NOT NULL,
  person_id       uuid REFERENCES bbl_people(id),
  name            text,
  given_name      text,
  family_name     text,
  roles           text[],
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

bbl_work_notes (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  val             text NOT NULL,
  kind            text,
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)
```

Same pattern for classifications, titles, abstracts, lay summaries, keywords.

### Cross-entity FK relations

Work-to-project, work-to-organization, work-to-work, person-to-organization
(affiliations), project-to-person (participants), organization-to-organization.

Each row has `id`, `*_source_id`, `user_id`, `asserted_at`, `pinned`. Pinned
as a collective — one asserter's full list is pinned.

```sql
bbl_work_projects (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  project_id      uuid NOT NULL REFERENCES bbl_projects(id),
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

bbl_work_organizations (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  org_id          uuid NOT NULL REFERENCES bbl_organizations(id),
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

bbl_work_rels (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  work_id         uuid NOT NULL REFERENCES bbl_works(id),
  related_work_id uuid NOT NULL REFERENCES bbl_works(id),
  kind            text NOT NULL,
  work_source_id  uuid REFERENCES bbl_work_sources(id),
  user_id         uuid REFERENCES bbl_users(id),
  asserted_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned          bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(work_source_id, user_id) = 1)
)

bbl_person_organizations (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  person_id         uuid NOT NULL REFERENCES bbl_people(id),
  organization_id   uuid NOT NULL REFERENCES bbl_organizations(id),
  valid_from        date,
  valid_to          date,
  person_source_id  uuid REFERENCES bbl_person_sources(id),
  user_id           uuid REFERENCES bbl_users(id),
  asserted_at       timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned            bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(person_source_id, user_id) = 1)
)

bbl_project_people (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id          uuid NOT NULL REFERENCES bbl_projects(id),
  person_id           uuid NOT NULL REFERENCES bbl_people(id),
  role                text,
  project_source_id   uuid REFERENCES bbl_project_sources(id),
  user_id             uuid REFERENCES bbl_users(id),
  asserted_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned              bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(project_source_id, user_id) = 1)
)

bbl_organization_rels (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id         uuid NOT NULL REFERENCES bbl_organizations(id),
  related_id              uuid NOT NULL REFERENCES bbl_organizations(id),
  kind                    text NOT NULL,
  organization_source_id  uuid REFERENCES bbl_organization_sources(id),
  user_id                 uuid REFERENCES bbl_users(id),
  asserted_at             timestamptz NOT NULL DEFAULT transaction_timestamp(),
  pinned                  bool NOT NULL DEFAULT false,
  CHECK (num_nonnulls(organization_source_id, user_id) = 1)
)
```

### Other entity fields

People, projects, organizations follow the same `*_fields` pattern as works.

- **Organization kind**: regular scalar assertion in `bbl_organization_fields`
  (no denormalized column, no profile — just a badge).
- **Organization names**: relation table `bbl_organization_titles` with language
  as grouping key — but pinned as a collective (one asserter's full list wins).
- **Project/organization dates** (`start_date`, `end_date`): regular scalar
  assertions in `bbl_project_fields` / `bbl_organization_fields`.

### Revs

```sql
bbl_revs (
  id         uuid PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  user_id    uuid REFERENCES bbl_users(id),
  source     text REFERENCES bbl_sources(id)
  -- both informational, both nullable
)
```

AddRev (user path): `user_id` set, `source` typically NULL.
Import: `source` set, `user_id` optional.
System batch: both can be NULL.

### Cache

`cache jsonb` on the entity table holds pinned values. Rebuilt on every
write from pinned rows across all assertion tables.

## Mutations

Every write — human or import — produces mutation records in `bbl_mutations`.
Replaying all mutations in order reproduces the current state of the database.

Mutations are concrete, named types. Two kinds per field:

- **Set** — sets the value. Expects a value; absence is an error for scalars.
  For collectives, expects at least one item (empty slice is an error).
- **Delete** — removes the assertion. Auto-pin re-evaluates.

Required fields have no Delete mutation (deleting them would always fail
validation):
- Work: titles (at least one)
- Person: name
- Project: titles (at least one)
- Organization: names (at least one)

### Entity lifecycle

- `CreateWork{Kind}` — creates entity row + initial assertions
- `DeleteWork{WorkID}` — sets status=deleted

### State mutations

`status` and `review_status` are not assertions. Simple state transitions:

```
SetWorkStatus{WorkID, Val}
SetWorkReviewStatus{WorkID, Val}
```

### Work mutations

Scalar fields (Set + Delete):

```
SetWorkArticleNumber / DeleteWorkArticleNumber
SetWorkBookTitle / DeleteWorkBookTitle
SetWorkConference / DeleteWorkConference
SetWorkEdition / DeleteWorkEdition
SetWorkIssue / DeleteWorkIssue
SetWorkIssueTitle / DeleteWorkIssueTitle
SetWorkJournalAbbreviation / DeleteWorkJournalAbbreviation
SetWorkJournalTitle / DeleteWorkJournalTitle
SetWorkPages / DeleteWorkPages
SetWorkPlaceOfPublication / DeleteWorkPlaceOfPublication
SetWorkPublicationStatus / DeleteWorkPublicationStatus
SetWorkPublicationYear / DeleteWorkPublicationYear
SetWorkPublisher / DeleteWorkPublisher
SetWorkReportNumber / DeleteWorkReportNumber
SetWorkSeriesTitle / DeleteWorkSeriesTitle
SetWorkTotalPages / DeleteWorkTotalPages
SetWorkVolume / DeleteWorkVolume
```

Collective fields (Set + Delete, except where noted):

```
SetWorkTitles                              (no delete — required)
SetWorkAbstracts / DeleteWorkAbstracts
SetWorkLaySummaries / DeleteWorkLaySummaries
SetWorkNotes / DeleteWorkNotes
SetWorkKeywords / DeleteWorkKeywords
SetWorkIdentifiers / DeleteWorkIdentifiers
SetWorkClassifications / DeleteWorkClassifications
SetWorkContributors / DeleteWorkContributors
```

### Person mutations

```
SetPersonName                              (no delete — required)
```

Other person fields follow the same Set/Delete pattern.

### Project mutations

```
SetProjectTitles                           (no delete — required)
```

Other project fields follow the same Set/Delete pattern.

### Organization mutations

```
SetOrganizationNames                       (no delete — required)
```

Other organization fields follow the same Set/Delete pattern.

### Write paths

Both human (AddRev) and import paths write the same mutation types to
`bbl_mutations`. They share the same low-level write helpers.

**Human path (AddRev):**
- Assertion rows get `user_id` set, `*_source_id = NULL`
- Replace semantics: DELETE existing human assertion + INSERT new one
- `bbl_revs` row with `user_id` set

**Import path:**
- Assertion rows get `*_source_id` set, `user_id = NULL`
- Re-import: DELETE all assertions for this source record + INSERT new ones
- `bbl_revs` row with `source` set

```
incoming record → evaluate
  ├─ high confidence → auto-accept
  │   1. Match to existing entity (or create new)
  │   2. UPSERT bbl_work_sources → get work_source_id
  │   3. DELETE all assertions WHERE work_source_id = $1
  │   4. INSERT new assertion rows with work_source_id set
  │   5. Write mutation records to bbl_mutations
  │   6. Auto-pin re-evaluates (human assertions untouched)
  │
  ├─ low confidence → auto-reject
  │
  └─ ambiguous → candidate
```

### What this replaces

| Old concept | New equivalent |
|---|---|
| `canWrite` | Gone — sources write their own assertions freely |
| `Provenance` / `FieldProvenance` | Gone — origin is `*_source_id` or `user_id` |
| `attrs jsonb` | Gone — scalars in `*_fields` table |
| `provenance jsonb` sidecar | Gone |
| `controlled_fields` | Gone — re-import just replaces source's own assertions |
| `mergeWorkFields` | Gone — no merge, just store assertions + auto-pin |
| `HasScalarUpdates` | Gone — compare against pinned value |
| `Field[T]` with `canWrite` | Row in `*_fields` with `pinned` bool |
| `pinned_by` | Gone — derived from `*_source_id` vs `user_id` |
