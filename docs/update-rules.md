# Update rules

## Context

With additive assertions, every unnecessary write creates a permanent row.
Every write path (form save, batch CSV, API) needs a consistent rule for
when to create assertions and when to skip.

## The rule

**Diff against pinned values. Only assert differences.**

When a user submits values (via form, CSV, or API), compare each submitted
value against the currently pinned value for that field. Only create
assertions for fields where the value actually differs.

- Same value as pinned → skip. Regardless of who pinned it.
- Different value → create assertion.
- Value cleared (was present, now absent) → Hide.

This rule is universal: same for users, curators, batch CSV, and API.
No per-user logic, no special cases.

## Why not per-user noop detection?

"Skip if the same user already asserted this value" doesn't work in
practice:

- User A edits a record, changes nothing, saves → assertions for every field.
- User B edits same record, changes nothing, saves → assertions for every field.
- Two curators save the same record without changes → both create assertions.

Per-user checks prevent self-duplication but not cross-user duplication.
The table still grows from unchanged saves.

## Why diff against pinned?

The pinned value is what the user sees in the form. If they submit a value
identical to what they saw, they changed nothing — no assertion needed.
If they submit something different, they're making a deliberate change.

This works because the form renders the pinned values. The diff catches
exactly the user's intent: what did they change from what they were shown?

## Blessing / review is separate

A curator opening a record and saving without changes does NOT create
assertions. If a curator wants to explicitly endorse a record's values
("I've reviewed this, it's correct"), that's a separate action — a review
status change, not an assertion. Conflating "save" with "bless" overloads
the form save with implicit semantics.

Assertions record opinions about field values.
Review records a judgment about the record as a whole.

## Architecture: policies, not baked-in rules

`Update()` is a dumb writer. It receives updaters and executes them —
no filtering, no opinions, no noop detection.

The diff logic lives in **update policies** — composable functions that
filter updaters before they reach `Update()`. Each caller picks the policy
appropriate for its context.

```go
// An UpdatePolicy filters updaters based on current entity state
// AND the full assertion landscape for the entity.
type UpdatePolicy func(entity any, assertions []Assertion, updaters []updater) []updater
```

Policies receive both the entity (cache = pinned values) and the current
assertions for the entity. The cache shows *what won*. The assertions show
*who said what* — the full structural picture. This allows policies to
reason about:

- Whether the pinned value matches the submitted value (diff)
- Whether the current user already has an identical assertion (self-dedup)
- Whether the field is curator-asserted (precedence awareness)
- Whether another user has a competing assertion (conflict context)

### Built-in policies

**DiffPolicy** — skip updaters where the value matches the pinned value.
This is the standard policy for form saves and batch CSV.

**NoopPolicy** — pass everything through. For callers who know exactly
what they want to assert (explicit API use, internal system operations).

Future policies can use the assertion data for richer behavior:
- **PrecedencePolicy** — skip if the current user can't outrank the
  existing pinner (e.g. user asserting over a curator-pinned field).
- **DeduplicationPolicy** — skip if the current user already has an
  identical assertion for this field (true self-dedup, not just pinned
  value comparison).

### Usage

```go
// Form handler
work, assertions := repo.GetWorkWithAssertions(ctx, workID)
updates := buildWorkUpdates(r, profile, work)
updates = DiffPolicy(work, assertions, updates)
if len(updates) > 0 {
    repo.Update(ctx, userID, updates...)
}

// Batch CSV
work, assertions := repo.GetWorkWithAssertions(ctx, workID)
updates := buildUpdatesFromCSV(row)
updates = DiffPolicy(work, assertions, updates)
// + conflict detection via rev_id (orthogonal to diff)
if len(updates) > 0 {
    repo.Update(ctx, userID, updates...)
}

// API — caller sent explicit updaters, execute as-is
repo.Update(ctx, userID, updates...)
```

### Why policies are separate from Update()

- `Update()` stays simple: validate, apply, write. One job.
- Policies are testable in isolation — pure functions, no DB.
- Different callers compose the pipeline they need.
- New policies (dry run, curator override, audit-only) are just new
  functions with the same signature.
- No implicit behavior surprises — the caller explicitly opts in.

## matches() method on updaters

Each updater implements `matches()` to support policy filtering:

```go
type updater interface {
    name() string
    needs() *updateNeeds
    apply(state updateState, userID *ID) (*updateEffect, error)
    write(ctx context.Context, tx pgx.Tx, revID int64) error
    matches(entity any) bool  // true = noop, skip this updater
}
```

`matches()` is pure — it compares against the entity's cached pinned
values. No DB access.

### Scalar fields

```go
func (m *SetWorkVolume) matches(w *Work) bool {
    return w.Volume == m.Val
}
```

### Collective fields

```go
func (m *SetWorkTitles) matches(w *Work) bool {
    if len(w.Titles) != len(m.Titles) { return false }
    for i := range m.Titles {
        if w.Titles[i] != m.Titles[i] { return false }
    }
    return true
}
```

### Hide

Noop if the field is already absent (no pinned value exists).

### Unset

Always apply — DELETE is idempotent.

### Lifecycle (Create/Delete)

Unchanged. Create always applies. Delete already handles noops.

## needs() change

Every updater declares its target entity IDs through `needs()`, returning
`*updateNeeds` (pointer — nil for zero-entity updaters).

```go
func (m *SetWorkVolume) needs() *updateNeeds {
    return &updateNeeds{workIDs: []ID{m.WorkID}}
}
```

`Update()` gathers all entity IDs from `needs()`, deduplicates, and
batch-fetches full entities once.

## UI: read-only for outranked fields

The form renders curator-pinned fields as read-only for non-curator users.
This prevents the ambiguity of submitting values for fields the user can't
outrank. The server never receives those fields, so no assertion is created.

The API does not enforce this — the caller is responsible for understanding
precedence.

## Batch CSV

The `rev_id` column handles conflict detection (someone changed the field
between export and upload). The diff policy handles noop detection (user
didn't change the field from what was exported). These are orthogonal
concerns.

## Deferred

- **Precedence policy:** what happens when a user submits a different value
  for a curator-pinned field via API. Recorded but doesn't change pin.
  Separate concern — could be a future policy.
- **Review/lock mechanism:** explicit curator endorsement of a record.
  Separate workflow, not an assertion.
