# Assertion Model

## System requirements

1. Sources and human assertions coexist side by side
2. A unified view of the entity is always available
3. Sources can re-import without overwriting human assertions
4. We only write when something new is said
5. A complete linear edit history is available per entity
6. Subsequent changes by the same actor are subsumed into one
7. Users can't overwrite curators
8. Sources can't overwrite users and curators
9. Interesting information about one field can come from multiple sources

## Rules

### Everything is an assertion

Every field value is an assertion. Assertions coexist side by side.
Each asserter maintains its own assertion independently.

An asserter is either a **source** (automated data feed, identified by a
source record in `bbl_*_sources`) or a **human** (user or curator,
identified by `user_id`). Humans are not sources -- `bbl_sources` only
contains automated data sources.

### Unified assertions table

Each entity type has a single assertions table (`bbl_work_assertions`,
`bbl_person_assertions`, etc.) that tracks who said what about which field.

- **Scalar fields**: one assertion row per asserter per field. Value stored
  inline (`val jsonb`).
- **Collection items**: one assertion row per item. Value stored inline
  (`val jsonb`). Position and sort order tracked per row.

There are no separate relation tables for pure-value collectives (titles,
keywords, identifiers, etc.). Collection items that need FKs (contributors
with `person_id`, work-project links, etc.) have a thin extension table
with the FK columns only.

### Replace semantics for human assertions

There is **at most one human assertion** per field (scalar) or **one set
of human items** per collective field. A human Set replaces the previous
value. There is no accumulation of assertion rows.

History is captured in a separate **history table** before the replace.

This keeps the assertion table compact. No GC needed, no DiffPolicy
needed to prevent bloat. A no-op UPSERT is harmless.

### History table for history

```sql
bbl_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id      bigint NOT NULL REFERENCES bbl_revs(id),
    record_type text NOT NULL,
    record_id   uuid NOT NULL,
    field       text NOT NULL,
    val         jsonb,          -- previous value (scalar or JSON snapshot of items)
    hidden      bool
)
```

Before a human Set or Hide replaces an existing assertion, the old value
is logged. Before an Unset deletes an assertion, the old value is logged.
Source re-imports do NOT log to the audit table -- source history is in
the source records (`*_sources.record bytea`).

The history table is append-only. No indexes beyond `(record_type, record_id)`.
No pinning, no assertions, no policies. Just facts about what was replaced.

Linear history for an entity: query the history table + current assertions,
ordered by rev_id.

### Assertion rows track their origin

Each assertion has exactly one of:
- `*_source_id` -- FK to the entity's `*_sources` table (source assertion)
- `user_id` -- FK to `bbl_users` (human assertion)

A `CHECK (num_nonnulls(*_source_id, user_id) = 1)` enforces this.

### Three operations

Every field supports exactly three operations:

**Set** -- assert a value. For scalars: UPSERT the assertion row. For
collectives: replace the asserter's items (delete old, insert new).
Value stored inline (`val jsonb`).

**Hide** -- assert that the field has no value. UPSERT a single assertion
row with `hidden = true, position = NULL`. For collectives, any existing
items from this asserter are deleted. If pinned, nothing is displayed
regardless of what other asserters say.

**Unset** -- withdraw the assertion. Delete the asserter's assertion
row(s). CASCADE deletes extension rows. Auto-pin re-evaluates.

| Operation | Assertion row | Value | Pinned behavior |
|---|---|---|---|
| **Set** | exists, `hidden=false` | `val` inline | Display the value(s) |
| **Hide** | exists, `hidden=true` | none | Display nothing (intentional) |
| **Unset** | removed | removed | Next asserter's values display |

### Pinning selects the display value

Pinning is always implicit -- a side effect of writes, never an explicit
operation.

### Two pinning modes

Hardcoded per field type in the Go field catalog:

- **exclusive** (default): one asserter wins. For scalars, one row pinned.
  For collections, all items from the winning asserter are pinned.
  Used for: all scalars, contributors, titles, abstracts, notes, keywords.
- **union**: items from all asserters are pinned. If the field type
  defines a dedup key, duplicates are collapsed (highest priority wins).
  If no dedup key, all items pinned as-is.
  Used for: identifiers, classifications.

### Copy-on-write

When a human asserts a collective field for the first time, the pinned
list is copied as new assertion rows under the human's user_id. The
human's copy can then be modified in place (add, remove, reorder).

Subsequent edits by the same human modify their existing rows directly.
No copy needed -- they already own the list.

### Auto-pin rule

1. Human assertion exists → pin it
2. No human assertion → highest-priority source wins
3. No assertions → field absent

There is at most one human assertion per field. The `role` column records
who asserted it (curator or user) but does not affect pinning — there's
only one human row to consider.

Source priority comes from `bbl_sources.priority`.

For **exclusive** fields: one asserter's rows get `pinned = true`.
For **union** fields: all asserters' rows get `pinned = true`.

### Curator lock

A curator assertion acts as a lock. Since there is only one human
assertion row per field, a curator's Set physically replaces any
previous human assertion (user or curator). Consequences:

- Users cannot assert a field that a curator has asserted (rejected as
  a permissions error in Update)
- If the curator Unsets, there is no user assertion to fall back to —
  it was already replaced. The field falls back to source.
- A curator can always overwrite another curator's assertion.
- A user can only assert fields where no curator assertion exists.

### Unset behavior

1. Remove the asserter's assertion rows (CASCADE deletes extension rows)
2. Auto-pin re-evaluates
3. Other assertions exist → next best asserter wins
4. No other assertions → field absent
5. Cache rebuilt

### Re-import

Delete all of this source record's assertions + insert new ones.
Human assertions untouched.

```sql
DELETE FROM bbl_work_assertions WHERE work_source_id = $1;
-- Insert new assertion rows (scalars + collection items)
INSERT INTO bbl_work_assertions (...) VALUES (...);
-- Insert extension rows for FK-bearing items
INSERT INTO bbl_work_assertion_contributors (...) VALUES (...);
```

No rows = no opinion. If a source doesn't mention a field, no assertion
rows are created. Hide is always an explicit action.

### Validation

Two layers:
- **Structural**: every assertion is validated on write (correct type,
  valid scheme, etc.). `CHECK (hidden OR val IS NOT NULL)` at DB level.
- **Completeness**: only pinned values count. Checked when status → public.

## Schema

### Assertion table (per entity type)

```sql
CREATE TABLE bbl_work_assertions (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id           bigint NOT NULL REFERENCES bbl_revs(id),
    work_id          uuid NOT NULL REFERENCES bbl_works(id) ON DELETE CASCADE,
    field            text NOT NULL,
    val              jsonb,
    hidden           bool NOT NULL DEFAULT false,
    work_source_id   uuid REFERENCES bbl_work_sources(id) ON DELETE CASCADE,
    user_id          uuid REFERENCES bbl_users(id) ON DELETE SET NULL,
    role             text,
    asserted_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned           bool NOT NULL DEFAULT false,
    position         text,         -- NULL for scalars, fracdex for collection items
    CHECK (field <> ''),
    CHECK (hidden OR val IS NOT NULL OR position IS NOT NULL),
    CHECK (num_nonnulls(work_source_id, user_id) = 1)
);

-- Source uniqueness (scalars + collection items)
CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field, work_source_id, position)
    WHERE work_source_id IS NOT NULL;

-- Human uniqueness (one human assertion per scalar field)
CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field)
    WHERE user_id IS NOT NULL AND position IS NULL;

-- Pinned value lookup (scalars + collection items)
CREATE INDEX ON bbl_work_assertions (work_id, field, position)
    WHERE pinned = true;
```

Same pattern for `bbl_person_assertions`, `bbl_project_assertions`,
`bbl_organization_assertions`.

### What assertion rows look like

**Scalars:**
```
field='volume'  val='"42"'                                       position=NULL
field='pages'   val='{"start":"101","end":"115"}'                position=NULL
```

**Pure-value collection items (inlined):**
```
field='titles'       val='{"lang":"eng","val":"My Paper"}'       position=0  position="a"
field='titles'       val='{"lang":"fra","val":"Mon Article"}'    position=1  position="b"
field='identifiers'  val='{"scheme":"doi","val":"10.1234/..."}'  position=0  position="a"
field='keywords'     val='"machine learning"'                    position=0  position="a"
```

**FK-bearing collection items (assertion row + extension):**
```
-- Assertion row
field='contributors' val='{"name":"Smith","given_name":"John","roles":["author"]}' position="a"
-- Extension row
bbl_work_assertion_contributors: assertion_id=X, person_id=P1
```

**Absence:**
```
field='volume'    hidden=true  val=NULL  position=NULL   -- scalar absence
field='keywords'  hidden=true  val=NULL  position=NULL   -- collective absence
```

### Extension tables

Only for collection items that need FK columns. 1:1 with an assertion row.

```sql
CREATE TABLE bbl_work_assertion_contributors (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions(id) ON DELETE CASCADE,
    person_id       uuid REFERENCES bbl_people(id) ON DELETE SET NULL,
    organization_id uuid REFERENCES bbl_organizations(id) ON DELETE SET NULL
);

CREATE TABLE bbl_work_assertion_projects (
    assertion_id  bigint PRIMARY KEY REFERENCES bbl_work_assertions(id) ON DELETE CASCADE,
    project_id    uuid NOT NULL REFERENCES bbl_projects(id) ON DELETE CASCADE
);

CREATE TABLE bbl_work_assertion_organizations (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations(id) ON DELETE CASCADE
);

CREATE TABLE bbl_work_assertion_rels (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions(id) ON DELETE CASCADE,
    related_work_id uuid NOT NULL REFERENCES bbl_works(id) ON DELETE CASCADE,
    kind            text NOT NULL,
    CHECK (kind <> '')
);
```

Same pattern for:
- `bbl_person_assertion_organizations` (organization_id, valid_from, valid_to)
- `bbl_project_assertion_people` (person_id, role)
- `bbl_organization_assertion_rels` (rel_organization_id, kind, start_date, end_date)

### position (fracdex)

`position text` is a fractional index (fracdex) for ordering collection
items. NULL for scalars. Currently always written as a fresh dense range
("a", "b", "c") on every Set or re-import. The text type future-proofs
for in-place edits (insert between "b" and "c" → "bm") without rewriting
the whole list.

**On creation (import or human Set):** assigned sequentially. Source A
imports 3 identifiers: "a", "b", "c". Later source B imports 2: "d", "e".

**On re-import:** old source rows deleted. New rows appended after current
max for other asserters' pinned items.

**On human replace:** delete old items, insert new ones with fresh
positions. Currently always a full rewrite.

### Sources, source records, entity tables, revs

Unchanged from the current schema. See `migrations/00001_schema.sql`.

### Cache

`cache jsonb` on the entity table holds pinned values. Rebuilt from:

```sql
SELECT field, val, position
FROM bbl_work_assertions
WHERE work_id = $1 AND pinned = true AND NOT hidden
ORDER BY field, position
```

For FK-bearing items, LEFT JOIN extension table:

```sql
SELECT a.field, a.val, a.position, c.person_id
FROM bbl_work_assertions a
LEFT JOIN bbl_work_assertion_contributors c ON c.assertion_id = a.id
WHERE a.work_id = $1 AND a.pinned = true AND NOT a.hidden
  AND a.field = 'contributors'
ORDER BY a.position
```

## Updates

### Operations

Concrete, named types per field. Three kinds:

- **Set** -- UPSERT the value. Replaces in place for humans. History table
  captures the old value before replace.
- **Hide** -- UPSERT with `hidden = true`. History table captures old value.
- **Unset** -- DELETE. History table captures old value. Auto-pin re-evaluates.

Required fields have no Hide or Unset:
- Work: titles (at least one)
- Person: name
- Project: titles (at least one)
- Organization: names (at least one)

### UI mapping

| User action | Update | Effect |
|---|---|---|
| Save a value | Set | UPSERT assertion, history table old value |
| Clear / empty a field | Hide | UPSERT hidden=true, history table old |
| "Reset to source" | Unset | DELETE assertion, history table old, auto-pin |

### Wire format

```json
{"create": "work", "work_id": "01J...", "kind": "journal_article"}
{"delete": "work", "work_id": "01J..."}
{"set": "work_volume", "work_id": "01J...", "val": "42"}
{"hide": "work_volume", "work_id": "01J..."}
{"unset": "work_volume", "work_id": "01J..."}
```

Five verbs, one shape.

### Write paths

**Human path (Update):**

```go
repo.Update(ctx, userID,
    &SetWorkVolume{WorkID: id, Val: "42"},
    &SetWorkPublisher{WorkID: id, Val: "Acme"},
    &UnsetWorkEdition{WorkID: id},
)
```

- History table captures old values before replace
- UPSERT assertion rows (user_id set, role set)
- Unset: DELETE + auto-pin re-evaluates
- `bbl_revs` row with `user_id` set

**Import path:**

- DELETE all assertions for this source record + INSERT new ones
- No history table (source history is in source records)
- `bbl_revs` row with `source` set

### Update policies

`Update()` is a simple writer. Policies are composable functions that
filter updaters before they reach `Update()`.

```go
type UpdatePolicy func(entity any, assertions []Assertion, updaters []updater) []updater
```

Policies receive both the entity (cache = pinned values) and the current
assertions (who said what — the full structural picture).

**DiffPolicy** — skip updaters where the value matches the pinned value.
Standard policy for form saves and batch CSV. Prevents unnecessary UPSERTs.

With replace semantics, a no-op UPSERT is harmless (just overwrites with
the same value) but DiffPolicy avoids the history table entry and rev creation.

### Review is separate from assertion

Saving a form without changes does NOT create assertions. Endorsing a
record ("I've reviewed this, it's correct") is a review status change,
not an assertion.

## History

Query the history table + current assertions:

```sql
-- Current state
SELECT field, val, hidden, user_id, role, asserted_at
FROM bbl_work_assertions
WHERE work_id = $1 AND user_id IS NOT NULL;

-- Previous values
SELECT field, old_val, old_hidden, user_id, created_at
FROM bbl_history
WHERE record_type = 'work' AND record_id = $1
ORDER BY id DESC;
```

Combined gives a complete linear history per entity.

For collectives, the history table stores a JSON snapshot of the item list
in `old_val` — one row per replaced collective, not one row per item.

## Batch edit

CSV-based batch editing for scalar fields. Each row carries a `rev_id`
column — the rev_id at export time. On upload, per-field conflict
detection compares the pinned assertion's rev_id against the exported
rev_id.

- `rev_id` of pinned assertion <= exported `rev_id` → safe to apply
- `rev_id` of pinned assertion > exported `rev_id` → conflict, skip

Collective fields use separate CSV files per type (stage 2).
