# Assertion Model

## Rules

### Everything is an assertion

Every field value is an assertion. Assertions coexist side by side. Nothing is
overwritten -- each asserter maintains its own assertion independently.

An asserter is either a **source** (automated data feed, identified by a source
record in `bbl_*_sources`) or a **human** (user or curator, identified by
`user_id`). Humans are not sources -- `bbl_sources` only contains automated
data sources.

### Unified assertions table

Each entity type has a single assertions table (`bbl_work_assertions`,
`bbl_person_assertions`, etc.) that tracks who said what about which field.

- **Scalar fields**: the value is stored inline in the assertion row (`val
  jsonb`).
- **Collective fields** (identifiers, contributors, titles, etc.): `val` is
  NULL on the assertion row. The actual values live in relation tables with an
  `assertion_id` FK back to the assertion.

This means there is no separate `*_fields` table. The assertions table absorbs
that role.

### Every assertion has an ID

Each assertion is a row with a bigint primary key from a shared sequence
(`bbl_assertion_seq`). The sequence provides global ordering across all
assertion tables -- the highest ID is the most recent assertion. This
ordering drives the auto-pin rule (most recent human wins).

### Assertion rows track their origin

Each assertion has exactly one of:
- `*_source_id` -- FK to the entity's `*_sources` table (source assertion)
- `user_id` -- FK to `bbl_users` (human assertion)

A `CHECK (num_nonnulls(*_source_id, user_id) = 1)` enforces this. Source
identity (which source, which priority) is derived by joining through the
source record -- no `source` column on assertion tables.

### Three operations

Every field supports exactly three operations:

**Set** -- assert a value. Creates a new assertion row with `hidden = false`.
For scalars, the value is stored inline (`val`). For collectives, value rows
are stored in the relation table with FK to the assertion. For human
assertions, previous assertions for the same field stay (additive). For source
assertions, re-import replaces all source assertions.

**Hide** -- assert that the field has no value. Creates a new assertion row
with `hidden = true`. The assertion row exists and can be pinned -- it means
"this field intentionally has no value." For collectives, the asserter's list
is explicitly empty. If pinned, no values are displayed regardless of what
sources assert.

**Unset** -- withdraw the assertion. Deletes the asserter's assertion row.
CASCADE deletes associated value rows in relation tables. The asserter no
longer has an opinion about this field. Auto-pin re-evaluates -- the next-best
asserter wins.

| Operation | Assertion row | Value | Pinned behavior |
|---|---|---|---|
| **Set** | exists, `hidden=false` | `val` (scalar) or relation rows (collective) | Display the value(s) |
| **Hide** | exists, `hidden=true` | none | Display nothing (intentional) |
| **Unset** | removed | removed | Next asserter's values display |

### Pinning selects the display value

One assertion per field is **pinned** -- the value used for display,
search, and export. All other assertions remain stored but are not displayed.

Pinning is always implicit -- a side effect of writes, never an explicit
operation.

### One kind of pin: the whole field

There is only one pinning granularity: the whole field. No additive, no
keyed, no exceptions.

- **Scalar**: grouping key = `(entity_id, field)`. One value wins.
- **Collective** (identifiers, contributors, titles, abstracts, notes,
  keywords, classifications, FK relations): grouping key = `(entity_id)` per
  table. One asserter's entire list wins.

### Copy-on-write: human assertion = human pin

When a human asserts a value (including selecting an existing source value),
they create their own assertion. The pin is on the human's assertion, never
on a source's.

For collectives, copy-on-write means the human's entire list is copied. A
human touches a list of 3000 authors -- it is now a full copy under the
human's user_id. Not storage efficient, but crystal clear.

Copy-on-write is generalized: not only over source assertions but also over
other humans. Each human edit creates a new assertion row -- it never
modifies or deletes an existing one (except Unset, which is an explicit
retraction).

Consequences:
- Source assertions can only be auto-pinned or not pinned
- Human assertions always win over source assertions
- Re-import freely replaces source assertions without breaking human pins
- Human assertion history is built up naturally from the additive rows

### Additive semantics for human assertions

Human assertions are purely additive. Each edit creates a new assertion row.
Multiple human assertions can coexist for the same (entity, field). The most
recent one with the highest role priority wins (see auto-pin rule).

Previous human assertions remain in the table -- they *are* the history.
Unset (retract) is the only operation that removes a human assertion row.

### Auto-pin rule

Uniform for all field types. Priority order:

1. **Recent curator** > **curator** > **recent user** > **user** > **source by priority**
2. No assertions → field absent

Within the same role level, the most recent assertion wins (highest
`id` from the shared assertion sequence). The role is stored on the
assertion row at assertion time (`role text`) -- not looked up live.

Source priority comes from `bbl_sources.priority`, looked up by joining
through `*_sources`.

If no human assertions exist, the highest-priority source wins. If no
assertions exist at all, the field is absent.

### Pin authority

No `pinned_by` column. Authority is derived from the assertion row:

- `*_source_id IS NOT NULL` → source assertion → can only be auto-pinned
- `user_id IS NOT NULL` → human assertion → wins over source
- `role` on the assertion row → determines priority among humans

### Unset behavior

Always the same:
1. Remove the asserter's assertion row (CASCADE deletes relation rows)
2. Auto-pin re-evaluates for that grouping key
3. Other assertions exist -> highest priority source gets auto-pinned
4. No other assertions -> field absent
5. Cache rebuilt

### Re-import

Re-import = delete all of this source record's assertions for the entity +
insert new ones. Source records are identified by `*_sources.id`.

```sql
-- Delete all assertions for this source record (CASCADE deletes relation rows)
DELETE FROM bbl_work_assertions WHERE work_source_id = $1;

-- Insert new assertions
INSERT INTO bbl_work_assertions (...) VALUES (...);
-- Insert collective values with FK to assertion
INSERT INTO bbl_work_contributors (...) VALUES (...);
```

- Auto-pinned fields: auto-pin re-evaluates
- Human-asserted fields: untouched (pin is on human assertion)

If a source previously asserted a field and the new import doesn't mention it,
the source's assertion is deleted (full replace). If a human had retracted,
the field becomes absent. This is correct: re-import is a full snapshot of
what the source currently knows.

### Contributors

Contributors are a collective -- each source asserts an ordered list, not
individual rows. Auto-pin picks the highest-priority source's entire list.
All source lists coexist in storage.

When a human edits contributors, they get a full copy-on-write of the list.
Individual rows within the human copy can be created/updated/deleted, but
pinning always operates on the full list.

### Validation

Two layers:
- **Structural**: every assertion is validated on write (correct type,
  valid scheme, etc.)
- **Completeness**: only pinned values count. Checked when status -> public.

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
sources -- humans are not in this table.

### Source records

Each entity type has a `*_sources` table with a synthetic `id` PK:

```sql
bbl_work_sources (
  id           uuid PRIMARY KEY,
  work_id      uuid NOT NULL REFERENCES bbl_works(id) ON DELETE CASCADE,
  source       text NOT NULL REFERENCES bbl_sources(id),
  source_id    text NOT NULL,
  candidate_id uuid REFERENCES bbl_work_candidates(id) ON DELETE SET NULL,
  record       bytea NOT NULL,
  fetched_at   timestamptz NOT NULL DEFAULT transaction_timestamp(),
  ingested_at  timestamptz NOT NULL DEFAULT transaction_timestamp(),
  UNIQUE (work_id, source, source_id)
)
```

Same pattern for `bbl_person_sources`, `bbl_project_sources`,
`bbl_organization_sources`.

### Entity tables

```sql
bbl_works (
  id            uuid PRIMARY KEY,
  version       int NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
  created_by_id uuid REFERENCES bbl_users(id) ON DELETE SET NULL,
  updated_by_id uuid REFERENCES bbl_users(id) ON DELETE SET NULL,
  kind          text NOT NULL,
  status        text NOT NULL,       -- 'private' | 'restricted' | 'public' | 'deleted'
  review_status text,                -- NULL | 'pending' | 'in_review' | 'returned'
  delete_kind   text,                -- 'withdrawn' | 'retracted' | 'takedown'
  deleted_at    timestamptz,
  deleted_by_id uuid REFERENCES bbl_users(id) ON DELETE SET NULL,
  cache         jsonb NOT NULL DEFAULT '{}'
)
```

`version` is for optimistic concurrency -- bumped on every write.

`created_by_id`/`updated_by_id` are cached from the first/last rev's `user_id`
for single-row entity reads.

`status` and `review_status` are state columns, not sourced fields. They
are not assertions and do not participate in pinning. They have their own
updates (`SetWorkStatus`, `SetWorkReviewStatus`).

`kind` is a regular assertion in `bbl_work_assertions` -- same pinning rules as
any other field. But it is structurally important: the pinned kind determines
the active profile (which fields are valid, which are required).

Special rule: **on entity creation, the system creates its own kind assertion
(copy-on-write from the source) and pins it.** This prevents kind from ever
changing silently on re-import. If a source later asserts a different kind,
it's visible for curation but the entity's effective kind stays stable.
A curator can change kind by creating their own assertion. Kind is cached as
the `kind` column in the entity table.

When pinned kind changes, fields valid under the old profile are not deleted
-- they become inactive (still stored, not validated for completeness).

### Assertions table (per entity type)

```sql
CREATE SEQUENCE bbl_assertion_seq;

CREATE TABLE bbl_work_assertions (
    id             bigint PRIMARY KEY DEFAULT nextval('bbl_assertion_seq'),
    rev_id         bigint NOT NULL REFERENCES bbl_revs (id),
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    field          text NOT NULL,
    val            jsonb,              -- scalar value; NULL for collective fields
    hidden         bool NOT NULL DEFAULT false,
    work_source_id uuid REFERENCES bbl_work_sources (id) ON DELETE CASCADE,
    user_id        uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role           text,               -- user role at assertion time (for pin priority)
    asserted_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned         bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(work_source_id, user_id) = 1)
);

-- Source: one assertion per source record per field
CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field, work_source_id)
    WHERE work_source_id IS NOT NULL;

-- One pin per field
CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field)
    WHERE pinned = true;
```

No unique constraint on human assertions -- multiple human assertions per
field are allowed (additive semantics). The auto-pin rule selects the winner
by role priority and recency.

Same pattern for `bbl_person_assertions`, `bbl_project_assertions`,
`bbl_organization_assertions`.

### Relation tables (collective fields)

Relation tables keep their domain-specific columns but carry only an
`assertion_id` FK for provenance tracking. No `*_source_id`, `user_id`,
`asserted_at`, or `pinned` -- all of that lives on the assertion row.

```sql
CREATE TABLE bbl_work_contributors (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    position       int NOT NULL,
    kind           text NOT NULL DEFAULT 'person' CHECK (kind IN ('person', 'organization')),
    person_id      uuid REFERENCES bbl_people (id) ON DELETE SET NULL,
    name           text NOT NULL,
    given_name     text,
    family_name    text,
    roles          text[] NOT NULL DEFAULT '{}'
);

CREATE TABLE bbl_work_identifiers (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme         text NOT NULL,
    val            text NOT NULL
);

CREATE TABLE bbl_work_keywords (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    val            text NOT NULL
);

CREATE TABLE bbl_work_notes (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    val            text NOT NULL,
    kind           text
);
```

Same pattern for classifications, titles, abstracts, lay summaries, and
cross-entity FK relations (work-project, work-organization, work-work,
person-organization, project-person, organization-organization).

### Determining pinned values

```sql
-- Scalar: value is inline
SELECT field, val
FROM bbl_work_assertions
WHERE work_id = $1 AND pinned = true AND hidden = false AND val IS NOT NULL;

-- Collective: join to relation table
SELECT c.*
FROM bbl_work_assertions a
JOIN bbl_work_contributors c ON c.assertion_id = a.id
WHERE a.work_id = $1 AND a.field = 'contributors' AND a.pinned = true AND a.hidden = false;

-- Absent fields: pinned but intentionally empty
SELECT field
FROM bbl_work_assertions
WHERE work_id = $1 AND pinned = true AND hidden = true;
```

### Other entity fields

People, projects, organizations follow the same assertions + relation table
pattern.

- **Organization kind**: regular scalar assertion in
  `bbl_organization_assertions` (no denormalized column, no profile -- just a
  badge).
- **Organization names**: relation table `bbl_organization_names` with
  `assertion_id` FK -- pinned as a collective (one asserter's full list wins).
- **Project/organization dates** (`start_date`, `end_date`): columns on the
  entity table, not assertions.

### Revs

```sql
bbl_revs (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  user_id    uuid REFERENCES bbl_users(id),
  source     text REFERENCES bbl_sources(id)
  -- both informational, both nullable
)
```

Update (user path): `user_id` set, `source` typically NULL.
Import: `source` set, `user_id` optional.
System batch: both can be NULL.

Every assertion row references the rev that created it via `rev_id`.

### Cache

`cache jsonb` on the entity table holds pinned values. Rebuilt on every
write from pinned assertion rows across the assertions table and all relation
tables. The `bbl_works_view` computes the cache via lateral joins against
pinned, non-hidden assertions.

## Updates

Updates are concrete, named types. Three kinds per field:

- **Set** -- sets the value. Expects a value; absence is an error for scalars.
  For collectives, expects at least one item (empty slice is an error).
  Creates a new assertion with `hidden = false`. Previous human assertions
  stay (additive).
- **Hide** -- asserts the field has no value. Creates a new assertion with
  `hidden = true`. Previous human assertions stay.
- **Unset** -- withdraws the assertion. Deletes the human assertion row(s),
  auto-pin re-evaluates.

Required fields have no Hide or Unset update (hiding/unsetting them would
always fail validation):
- Work: titles (at least one)
- Person: name
- Project: titles (at least one)
- Organization: names (at least one)

### UI mapping

| User action | Update | Assertion state |
|---|---|---|
| Save a value | Set | `hidden=false`, value stored |
| Clear / empty a field | Hide | `hidden=true`, no value |
| "Reset to source" | Unset | assertion removed, source wins |

### Entity lifecycle

- `CreateWork{Kind}` -- creates entity row + initial assertions
- `DeleteWork{WorkID}` -- sets status=deleted

### State updates

`status` and `review_status` are not assertions. Simple state transitions:

```
SetWorkStatus{WorkID, Val}
SetWorkReviewStatus{WorkID, Val}
```

### Work updates

Scalar fields (Set + Unset):

```
SetWorkArticleNumber / UnsetWorkArticleNumber
SetWorkBookTitle / UnsetWorkBookTitle
SetWorkConference / UnsetWorkConference
SetWorkEdition / UnsetWorkEdition
SetWorkIssue / UnsetWorkIssue
SetWorkIssueTitle / UnsetWorkIssueTitle
SetWorkJournalAbbreviation / UnsetWorkJournalAbbreviation
SetWorkJournalTitle / UnsetWorkJournalTitle
SetWorkPages / UnsetWorkPages
SetWorkPlaceOfPublication / UnsetWorkPlaceOfPublication
SetWorkPublicationStatus / UnsetWorkPublicationStatus
SetWorkPublicationYear / UnsetWorkPublicationYear
SetWorkPublisher / UnsetWorkPublisher
SetWorkReportNumber / UnsetWorkReportNumber
SetWorkSeriesTitle / UnsetWorkSeriesTitle
SetWorkTotalPages / UnsetWorkTotalPages
SetWorkVolume / UnsetWorkVolume
```

Collective fields (Set + Unset, except where noted):

```
SetWorkTitles                                (no unset -- required)
SetWorkAbstracts / UnsetWorkAbstracts
SetWorkLaySummaries / UnsetWorkLaySummaries
SetWorkNotes / UnsetWorkNotes
SetWorkKeywords / UnsetWorkKeywords
SetWorkIdentifiers / UnsetWorkIdentifiers
SetWorkClassifications / UnsetWorkClassifications
SetWorkContributors / UnsetWorkContributors
SetWorkProjects / UnsetWorkProjects
SetWorkOrganizations / UnsetWorkOrganizations
SetWorkRels / UnsetWorkRels
```

### Person updates

```
SetPersonName                                (no unset -- required)
SetPersonGivenName / UnsetPersonGivenName
SetPersonMiddleName / UnsetPersonMiddleName
SetPersonFamilyName / UnsetPersonFamilyName
SetPersonIdentifiers / UnsetPersonIdentifiers
SetPersonOrganizations / UnsetPersonOrganizations
```

### Project updates

```
SetProjectTitles                             (no unset -- required)
SetProjectDescriptions / UnsetProjectDescriptions
SetProjectIdentifiers / UnsetProjectIdentifiers
SetProjectPeople / UnsetProjectPeople
```

### Organization updates

```
SetOrganizationNames                         (no unset -- required)
SetOrganizationIdentifiers / UnsetOrganizationIdentifiers
SetOrganizationRels / UnsetOrganizationRels
```

### Write paths

**Human path (Update):**

```go
repo.Update(ctx, userID,
    &SetWorkVolume{WorkID: id, Val: "42"},
    &SetWorkPublisher{WorkID: id, Val: "Acme"},
    &UnsetWorkEdition{WorkID: id},
)
```

- Assertion rows get `user_id` set, `role` set, `work_source_id = NULL`
- Additive: new assertion row inserted, previous human assertions stay
- Unset: DELETE the human assertion row, auto-pin re-evaluates
- `bbl_revs` row with `user_id` set

**Import path:**
- Assertion rows get `work_source_id` set, `user_id = NULL`, `role = NULL`
- Re-import: DELETE all assertions for this source record + INSERT new ones
- `bbl_revs` row with `source` set

**CLI:**

```sh
# Human updates from stdin
echo '{"set":"work_volume","work_id":"01J...","val":"42"}' | bbl update --user 01J...

# Source import from stdin
cat records.jsonl | bbl works import plato
```

**Ingestion flow:**

```
incoming record -> evaluate
  +-- high confidence -> auto-accept
  |   1. Match to existing entity (or create new)
  |   2. UPSERT bbl_work_sources -> get work_source_id
  |   3. DELETE FROM bbl_work_assertions WHERE work_source_id = $1 (CASCADE)
  |   4. INSERT new assertion rows with work_source_id set
  |   5. INSERT collective values with assertion_id FK
  |   6. Auto-pin re-evaluates (human assertions untouched)
  |
  +-- low confidence -> auto-reject
  |
  +-- ambiguous -> candidate
```

## History

### How history works

Source assertions have no history in the assertion tables. Re-import
deletes all of a source's assertions and inserts new ones. The source
record (`*_sources.record bytea`) preserves the original payload if
needed.

Human assertions are additive — each edit creates a new assertion row.
Previous assertions stay in the table, unpinned. History for a field is
the sequence of human assertion rows ordered by `id` (from the shared
sequence):

```sql
-- History of human edits to volume on a work
SELECT id, rev_id, user_id, role, val, hidden, asserted_at
FROM bbl_work_assertions
WHERE work_id = $1 AND field = 'volume' AND user_id IS NOT NULL
ORDER BY id;
```

The named operation is derivable from each row:
- `val` present, `hidden = false` → **Set**
- `hidden = true` → **Hide**
- Row absent (was deleted) → **Unset** (retraction)

### Table growth

Human edits are rare — the assertion table grows slowly from human
history. Source re-imports produce no history (delete + insert).

Unpinned human assertions can be pruned by application logic without
affecting current state. Conservative GC rule: for the same (entity,
field, user, role), only the most recent assertion can ever win — older
ones from the same user at the same role level are safe to prune:

```sql
DELETE FROM bbl_work_assertions a
WHERE a.pinned = false
  AND a.user_id IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM bbl_work_assertions b
    WHERE b.work_id = a.work_id
      AND b.field = a.field
      AND b.user_id = a.user_id
      AND b.role = a.role
      AND b.id > a.id
  );
```

Assertions from different users or different roles are preserved — they
represent meaningful alternatives that could surface on retraction.
