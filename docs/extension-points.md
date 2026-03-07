# Extension points

This document describes the stable extension surface for bbl — the interfaces,
registries, and hooks that a custom binary can use to add or replace behaviour
without forking the core.

See `third-party-customization.md` for the broader customization philosophy and
layering. This document is the technical reference for Go developers building on
top of bbl as a library.

---

## General pattern

All extension points follow the same model: register before `app.Run()`, the app
discovers registrations at startup. There is no dynamic loading or plugin system.

```go
func main() {
    app := bbl.New(bbl.Config{ /* ... */ })

    app.RegisterWorkFormat("datacite", datacite.Format{})
    app.RegisterWorkSource("wos", wos.NewSource(cfg))
    app.RegisterAuthProvider(oidc.NewProvider("ugent_oidc", cfg))
    app.RegisterSlot("nav.extensions", myNavComponent)
    app.RegisterMutation("SetGrantNumber", grantmut.SetGrantNumber{})

    app.Run()
}
```

---

## Work formats

A work format encodes and optionally decodes works for a named scheme. The scheme
is the stable key used in `bbl_work_representations`, OAI-PMH metadata prefixes,
and export format names.

```go
type WorkEncoder interface {
    Encode(*Work) ([]byte, error)
}

// Optional: formats that can parse records from external sources.
type WorkDecoder interface {
    Decode([]byte) (*RawWorkCandidate, error)
}

// Optional: formats that need a collection wrapper (XML envelope, etc.).
type WorkStreamer interface {
    NewStream(io.Writer) WorkStream
}

type WorkStream interface {
    Add(*Work) error
    Close() error  // writes postamble, flushes
}

RegisterWorkFormat(scheme string, enc WorkEncoder)
```

At use time the app checks for optional interfaces:

```go
if dec, ok := enc.(WorkDecoder); ok { /* pull-on-demand, harvester decode */ }
if str, ok := enc.(WorkStreamer); ok { /* collection export */ }
```

**OAI-PMH** reuses the representation cache — it does not have its own encoders.
Registered formats are opted into OAI-PMH with an optional metadata prefix mapping:

```go
RegisterOAIPMHFormat(scheme string, metadataPrefix string)
// e.g. RegisterOAIPMHFormat("datacite_4", "oai_datacite")
```

OAI-PMH serves `bbl_work_representations` rows for whitelisted schemes. The
`updated_at` on each representation row is the OAI-PMH datestamp for that record
in that format — different formats can have different datestamps for the same work.

---

## Work sources (harvesters)

A work source yields raw candidate records from an external system. Scheduling is
handled by Catbird — the source has no knowledge of when it is called.

```go
type WorkSource interface {
    IdentifierSchemes() []string
    Harvest(context.Context) iter.Seq2[*RawWorkCandidate, error]
}

// Optional: sources that support pull-on-demand (e.g. fetch by DOI).
type WorkSourcePuller interface {
    WorkSource
    Pull(ctx context.Context, id string) (*RawWorkCandidate, error)
}

type RawWorkCandidate struct {
    SourceRecordID string
    Attrs          json.RawMessage
    Identifiers    []Identifier
    ExpiresAt      *time.Time   // nil = no expiry
    Confidence     *float64     // nil = unknown
}

RegisterWorkSource(id string, source WorkSource)
```

`id` matches `bbl_sources.id`. The ingest layer handles dedup, upsert into
`bbl_work_candidates`, identifier extraction, and person/org matching. The source
just yields raw records.

Pull-on-demand (user imports a work by ID) uses `WorkSourcePuller` if implemented.
The UI checks `_, ok := src.(WorkSourcePuller)` to decide whether to show the
import field for that source.

If the registered format for this source implements `WorkDecoder`, the source can
hand raw bytes to it for parsing rather than building `RawWorkCandidate` manually.

---

## User sources

A user source yields user records from an external directory. Scheduling is handled
by Catbird.

```go
type UserSource interface {
    IdentifierSchemes() []string
    Harvest(context.Context) iter.Seq2[*User, error]
}

RegisterUserSource(id string, source UserSource)
```

The ingest layer stamps `bbl_user_sources.last_seen_at` for each yielded user.
After the sweep, a Catbird job deactivates users absent from the sweep:

```sql
SELECT user_id FROM bbl_user_sources
WHERE source = $1
  AND expires_at IS NOT NULL
  AND last_seen_at < $sweep_started_at
```

`expires_at IS NULL` marks permanent rows (one-time imports, manually added users)
— the staleness sweep skips these.

The ingest layer also auto-associates the configured auth provider for this source,
writing a `bbl_user_auth_methods` row for each harvested user.

---

## Auth providers

Auth providers handle login flows. A user may have multiple auth methods; each
references a named provider instance, not a generic protocol type — so a user can
have both `ugent_oidc` and `orcid_oidc` simultaneously.

```go
type AuthProvider interface {
    ID() string   // stable name; matches bbl_user_auth_methods.provider
    BeginAuth(w http.ResponseWriter, r *http.Request) error
    CompleteAuth(w http.ResponseWriter, r *http.Request) (Claims, error)
}

RegisterAuthProvider(provider AuthProvider)
```

Login flow:
1. User selects a provider on the login page (one button per registered provider)
2. `BeginAuth` initiates the flow
3. `CompleteAuth` returns claims; the app looks up `bbl_user_auth_methods` by
   `(provider.ID(), claims.Identifier)` to find the user
4. Session created

Users must be pre-provisioned — no auto-creation on first login. Future: limited-
access auto-registration (e.g. ORCID-only users) follows the user candidate pattern.

---

## Mutations

A mutation is a named, serializable unit of state change. It replaces the
`Action`/`WorkChanger` split in the prototype — one concept, one registry, no
nesting.

```go
type Mutation struct {
    Name       string
    EntityType string
    EntityID   uuid.UUID
    Args       json.RawMessage
}

type MutationImpl interface {
    // Needs declares what state must be pre-fetched. Computable from args alone.
    Needs(m Mutation) MutationNeeds

    // Apply is pure: no DB access. Receives pre-fetched state, returns diff.
    Apply(state MutationState, m Mutation) (Diff, error)
}

type MutationNeeds struct {
    WorkIDs    []uuid.UUID
    PersonRefs []Ref
    // ...
}

type MutationState struct {
    Works   map[uuid.UUID]*Work
    Persons map[uuid.UUID]*PersonIdentity
    // ...
}

RegisterMutation(name string, impl MutationImpl)
```

`AddRev(ctx, userID, source, []Mutation, IndexMode)` applies a slice of mutations
in one transaction:

1. Call `Needs` on all mutations → union requirements
2. Full entity read for validation + state prefetch (one batch, pgx pipeline)
3. Build `MutationState`
4. Call `Apply` on each mutation → collect diffs
5. One batch write: entity updates + `bbl_mutations` rows
6. Post-commit: index according to `IndexMode`

One read round-trip + one write round-trip regardless of mutation count. `Apply`
is pure and testable without a DB connection.

Mutations are plain data — buildable inline, from JSON, from an API payload, or
by a harvester pipeline — and passed to `AddRev` unchanged.

`IndexMode` controls search indexing behaviour after commit:

```go
type IndexMode int

const (
    IndexSync  IndexMode = iota  // refresh=wait_for — GUI writes
    IndexAsync                   // Catbird job     — normal writes
    IndexSkip                    // no indexing     — bulk imports
)
```

---

## Search indexing

The work indexer is a registered component — the default OpenSearch implementation
ships with bbl; a custom binary can replace it:

```go
app.RegisterWorkIndexer(indexer WorkIndexer)

type WorkIndexer interface {
    Index(ctx context.Context, doc *WorkSearchDocument) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

`refresh` strategy (`wait_for` vs. async) is passed by the `AddRev` call site via
`IndexMode`, not part of the interface.

---

## Web app

Extensions interact with the HTTP layer through three mechanisms.

### Routes

```go
app.RegisterRoute(method, pattern, handler)
```

Registered before `app.Run()`. The router is open at startup. Extension handlers
own their UI entirely — they may use any rendering approach.

### Layout wrapper

Extension handlers that want to render inside the base chrome (nav, header, footer)
call the layout wrapper:

```go
layout := app.Layout()
// layout is a func(ctx, w, r, title, content templ.Component) error
```

### Slots

Base templates expose named injection points as `templ.Component` parameters.
Extensions register components to fill them:

```go
app.RegisterSlot(name string, component templ.Component)
```

`templ.Component` is an interface (`Render(ctx, w) error`) — any rendering
approach that implements it works: compiled templ, `html/template` wrapped,
raw writer. Built-in slots are filled by compiled templ components; extension
slots can use anything.

**Defined slots** (added as concrete needs arise, not speculatively):

| Slot | Location |
|---|---|
| `nav.extensions` | Navigation — extension links section |
| `work.form.extra` | Work edit form — after profile-driven fields |
| `work.detail.sidebar` | Work detail page — sidebar |

Field type registration includes a `templ.Component` renderer for custom form
widgets within the work form.

Extension UI is deliberately loosely coupled from the base UI. Extensions own
their pages; slots are the only injection points into base pages. This bounds
the risk of extension breakage affecting core UI.

---

## Open questions

- **Slot data contract**: slots that receive typed data (e.g. `*Work`) need a
  stable data shape. How is this typed — `any` with a cast, or a typed slot
  variant per data shape?
- **Multiple slot registrations**: can multiple extensions fill the same slot?
  If so, are they concatenated or does last-registration win?
- **MutationNeeds extensibility**: as new entity types are added, `MutationNeeds`
  and `MutationState` grow. How is this kept open without breaking registered
  `MutationImpl`s?
