# Plan: Noop detection in the write path

## Context

With additive assertions, every unnecessary write creates a permanent row.
Form saves, batch imports, and API calls all need to avoid self-duplication —
asserting a value the same user already asserted for the same field.

## Noop rule

**Noop = the same user already has an assertion with this exact value for
this field.** Prevents self-duplication. Does not compare against other
asserters' values or consider precedence.

## Design

### `needs()` change

Every updater declares its target entity IDs through `needs()`, returning
`*updateNeeds` (nil only for truly zero-entity updaters). This unifies
"I target this entity" and "I need this entity for apply()" — handles 0
or N entities naturally.

```go
func (m *SetWorkVolume) needs() *updateNeeds {
    return &updateNeeds{workIDs: []ID{m.WorkID}}
}
```

### Noop check in `write()`

Each write helper checks the DB before inserting. If the same user already
has an identical assertion for the field, skip the insert.

**Scalar fields:**
```go
func writeSetWorkField(ctx, tx, revID, workID, field, val, userID, role) error {
    var exists bool
    tx.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM bbl_work_assertions
            WHERE work_id = $1 AND field = $2 AND user_id = $3 AND val = $4
        )`, workID, field, userID, marshaledVal).Scan(&exists)
    if exists {
        return nil // noop
    }
    // ... insert
}
```

**Hide:** check if user already has `hidden = true` for this field.
**Unset:** DELETE is idempotent — no noop check needed.
**Collective fields:** find user's most recent assertion for the field,
load its relation rows, compare against submitted list. Skip if identical.
**Lifecycle:** unchanged (Create always applies, Delete already noops).

### Update() flow

1. Type-assert each arg to `updater`
2. Gather all entity IDs from `needs()` — deduplicate
3. `fetchUpdateState` fetches entities with `FOR UPDATE`
4. Apply all updaters — `apply(state, userID)` returns effects
5. If all nil → rollback
6. Insert `bbl_revs` row
7. Write each non-noop updater — **noop check in `write()`**
8. If nothing was actually written → rollback (no rev needed)
9. Auto-pin, cache rebuild as before

### UI: precedence-aware forms

The form renders curator-pinned fields as **read-only**. Users see the
value but can't submit changes for fields they can't outrank. This avoids
the ambiguity of silently recorded assertions that don't affect display.

The API records all assertions as-is — caller's responsibility.

### Impact on callers

**Form handler:** can drop the ad-hoc scalar guard. Emit updaters for
every editable form field. `write()` skips duplicates.

**Batch edit:** conflict detection (rev_id) stays. Noop detection
delegates to `write()` — no separate diff logic needed.

### Files to modify

| File | Changes |
|---|---|
| `updaters.go` | `needs()` returns `*updateNeeds` |
| `revs.go` | Aggregate `*updateNeeds`. Handle all-noop case after writes. |
| `work_field_updaters.go` | `needs()` returns entity ID. Noop check in write helpers. |
| `work_relation_updaters.go` | Same. Collective noop check. |
| `work_updaters.go` | `needs()` for Create. |
| `person_field_updaters.go` | Same pattern. |
| `person_updaters.go` | Same. |
| `project_field_updaters.go` | Same. |
| `project_updaters.go` | Same. |
| `organization_field_updaters.go` | Same. |
| `organization_updaters.go` | Same. |
| `app/edit_handlers.go` | Remove scalar guard. |

### Deferred

- **Precedence policy:** what happens when a user asserts over a
  curator-owned field via API. Separate concern from noop detection.
- **UI read-only rendering:** form templates render curator-pinned
  fields as read-only. Separate PR.

## Verification

1. `go build ./...` + `go test -count=1 ./...`
2. Save form without changes → no new assertion rows
3. Save with one field changed → exactly one new assertion
4. Batch import with no changes → "no changes to apply"
5. Same user re-asserts same value via API → no new row
