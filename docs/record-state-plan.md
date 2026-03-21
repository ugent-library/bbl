# Plan: Explicit record state (write model)

## Problem

The write side has no explicit model. Record state is scattered across
`updateState.works[id]`, `fieldValues[id]`, `fieldStates[id]`, and
ad-hoc preview maps. Set/Hide/Unset reach into global bags by ID.
Validation dispatches on entity type with hardcoded field knowledge.
Lifecycle updaters conflate command and query — they mutate full
entity objects in memory and return them to callers.

## What changes

1. One `recordState` struct for all 4 entity types.
2. All record types are profile-driven — person, project, organization
   get profiles alongside work.
3. Validation is profile-driven via `validateRecord`.
4. Set/Hide/Unset mutate `recordState.fields` directly.
5. `Update` has a clean linear lifecycle — commands in, effects out.
6. Lifecycle updaters are plain commands — no entity objects in state.
7. Read models for indexing are fetched separately, not carried through
   the write pipeline.

## Record state

✅ Implemented in `record_state.go`.

```go
type recordState struct {
    recordType string
    id         ID
    version    int
    status     string
    kind       string              // empty for person, project
    fields     map[string]any
    assertions map[string]*fieldState
}
```

## Unified profiles

✅ Implemented in `profile.go`.

```go
type Profiles struct {
    Work         map[string][]FieldDef // kind → fields
    Organization map[string][]FieldDef // kind → fields
    Person       []FieldDef            // flat (no kinds)
    Project      []FieldDef            // flat (no kinds)
}

type FieldDef struct {
    ft       *fieldType
    Name     string
    Type     string   // resolved fieldType name (for views)
    Required string   // "", "always", "public"
    Schemes  []string // for identifier, classification
}
```

`required` is a string, not a bool. `"always"` = display identity
(name, titles), `"public"` = required when publishing.

## Validation

✅ Implemented in `field_validation.go` and `field_type.go`.

`fieldType.validate` has the signature
`func(val any, def *FieldDef) []*vo.Error`. nil for types with no
domain rules. Domain validation runs on all present values, not just
required fields.

Implemented validators:
- `ftTitle`: non-blank val, valid ISO 639-2 lang
- `ftIdentifier`: non-blank scheme+val, scheme ∈ def.Schemes
- `ftWorkContributor`: each contributor has a name

## Set/Hide/Unset on record state

✅ Implemented. apply mutates `rs.fields` directly. After all applies,
`rs.fields` is the final state — no preview reconstruction needed.

## Update lifecycle

The `Update` method follows a clean linear flow. All state lives in
`map[ID]*recordState`. No entity object maps, no scattered bags.

### Signature

```go
func (r *Repo) Update(ctx context.Context, user *User, updates ...any) (bool, []RevEffect, error)
```

The caller passes the full `*User` — no DB lookup inside the
transaction. `user.ID` for attribution, `user.Role` for curator
lock checks. If future rules need more user context, it's there.

### Flow

```
 1. Parse updaters (type-assert to updater interface)
 2. Collect needs → single batch query:
    lock rows (SELECT ... FOR UPDATE) + fetch pinned field state
    → build map[ID]*recordState
 3. Apply all updaters (mutate recordState)
 4. If all noop → return (false, nil, nil)
 5. Validate (rs.fields + profile defs)
 6. Insert bbl_revs row
 7. Write all:
    - field ops via batch pipeline (executeFieldWrites)
    - lifecycle ops via their own write()
 8. Bump version + updated_at + updated_by_id
    for all affected existing records (records in state.records)
    — Create inserts with version=1, engine doesn't bump those
 9. Auto-pin for affected grouping keys
10. Rebuild cache for affected entities
11. Commit
12. Return (true, []RevEffect{RecordType, RecordID}, nil)
```

### updateState

```go
type updateState struct {
    records map[ID]*recordState
}
```

No entity maps. No fieldValues/fieldStates bags. No assertion info
maps. Everything lives on recordState.

### updateEffect

```go
type updateEffect struct {
    recordType string
    recordID   ID
    autoPin    func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error
}
```

No `record any`. Lifecycle updaters are commands — they write their
specific columns (status, delete_kind, deleted_at, etc.) and nothing
else. The engine owns version bumping and actor IDs.

### RevEffect

```go
type RevEffect struct {
    RecordType string
    RecordID   ID
    Version    int
}
```

Minimal. Callers that need read models (e.g. indexing) batch-read
from the repo. Version enables stale detection: read affected →
if entity.Version > effect.Version → another write happened, skip.
This matches the import path which already re-reads via
`EachWorkSince`.

### Indexing

`Services.UpdateAndIndex` calls `Update`, then batch-reads affected
entities from the repo for indexing. Same pattern as
`ImportWorksAndIndex`.

### Lifecycle updaters (Delete*)

Delete updaters become plain commands:
- `apply`: check `state.records[id].status` for noop
- `write`: `UPDATE ... SET status='deleted', delete_kind=...,
  deleted_at=now(), deleted_by_id=... WHERE id=...`
- No in-memory entity mutation, no entity returned in effect
- Version bump + updated_by_id handled by the engine

Create updaters stay self-contained — they build their INSERT
from their own struct fields, never read from state. New entities
don't exist in `state.records`, so the engine skips version bump.

### setRecordActorIDs

Deleted. Actor IDs are set in SQL:
- Lifecycle write() sets entity-specific actors (deleted_by_id)
- Engine version bump sets updated_by_id
- Create.write() sets created_by_id in the INSERT

## What was deleted (already done)

- `WorkProfiles`, `WorkKind`, `WorkFieldDef` types
- `fieldDef` struct, `resolveFieldDef`, `buildWorkFieldDefs`
- `validateFieldPreview`, `buildFieldPreview`, `entityMeta`
- Per-entity scan functions (`scanWorkUpdateRow`, etc.)
- `fieldValues`, `fieldStates`, `*Assertions` maps on updateState
- `assertionInfo` type, `fieldHidden`, `fieldCuratorLocked`
- `parseAssertionsInfo`, `logWorkHistory`, etc.

## Implementation status

All items implemented. Additional cleanup done beyond the original plan:

- `write()` returns `(string, []any)` instead of executing directly —
  lifecycle writes + version bumps batched in one `SendBatch`
- `logHistory`, `deleteHumanAssertions`, `insertAssertion` helpers
  removed (inlined in `executeFieldWrites` batch)
- All callers go through `UpdateAndIndex` for indexing
- `userID` helper in app removed
