# Greenfield core domain schema — design sketch

Iterative starting point. Meant for discussion, not direct migration.  
Reference schema: `pgxrepo/migrations/00001_initial_migration.sql`.

---

## Weaknesses in current schema

| Area | Issue |
|---|---|
| Organizations | `bbl_organization_rels` has no temporal bounds — can't express that A was part of B from 2010–2020 |
| Organizations | No merge/split history; reorganizations are destructive |
| Organizations | Identifiers can be reused across orgs over time with no validity window |
| People | Flat model, no identity/dedup layer (two-layer records/identities model) |
| People | `bbl_person_organizations` has no role or temporal affiliation |
| People | User↔person link is implicit (shared identifiers) — no explicit FK, no audit trail; identifier drift silently breaks or creates associations; `bbl_user_identifiers` conflates SSO login claims with authority matching signals |
| Identifiers | `idx`-ordered arrays everywhere; no source, no validity, no normalization |
| Work candidates | Modeled as works with a special status; pollutes the works table and indexes with large volumes of stale, low-quality rows; can't prune without losing rejection history |
| Work candidates | No source provenance on accepted works — once a candidate is accepted there is no way to trace which source system(s) contributed to a work or re-sync with them |
| Work contributors | No publication-time attribution snapshot; rendering a work requires joining to the current identity state, which may differ from what was recorded at time of entry (identity renamed, relinked, or deliberately unlinked by a curator) |
| Work contributors | No role distinction (author vs editor vs translator etc.) at schema level |
| Work files | No checksum, no upload status, no per-file access control |
| Representations | Hard FK to `bbl_works` — work-only; other entity types added when needed |
| Representations | No staleness signal — no way to know if a cached representation is still current without comparing to the entity's `updated_at` |
| Work permissions | No PK, no timestamps, no expiry |
| Permissions | Rights are scattered across `bbl_users.role`, `bbl_work_permissions`, and implicit creator logic — no single query can show what a user is allowed to do |
| Permissions | Global role (`admin \| curator \| user`) is unscoped — a curator has equal rights over all works and people; in practice curation is usually limited to a faculty or department |
| Permissions | Implied creator ownership is not schema-represented — can't be transferred, queried, or revoked without touching application logic |
| Permissions | Proxy delegation is all-or-nothing — user A gets full impersonation of user B with no way to scope it to a subset of their entities |
| Permissions | Ad-hoc grants (`bbl_work_permissions`) apply only to works — no equivalent for person records, projects, or other entities |
| User proxies | No temporal bounds, no reason |
| `bbl_revs` | No source/context field (which system made this change?) |
| `bbl_mutations` | `diff jsonb` is opaque; no `op_type` (create/update/delete) |
| `bbl_mutations` | SUM check constraint is fragile; no explicit `entity_type` discriminator |
| Projects | No temporal bounds at schema level (start/end dates only in attrs) |
| Lists | `bbl_list_items` hard-codes `work_id` — no support for lists of persons, organizations, or projects |
| Lists | A list has no declared type constraint — nothing prevents mixing entity types unless enforced in application code |
| General | No source registry / import precedence table |
| General | `idx int` ordering requires renumbering all following rows on any insertion or reorder — every user-reorderable list pays that cost |
| Work rels | Directional — querying all works related to X requires checking both `work_id = X` and `rel_work_id = X`; `kind` semantics (symmetric vs asymmetric) are undocumented at schema level |
| Works | No tombstone metadata — `status='deleted'` provides the tombstone but there is no `deleted_at` / `deleted_by_id` to record when or by whom a work was withdrawn or retracted; no `delete_kind` to distinguish routine withdrawal from a legally-mandated takedown (GDPR, patent, right to be forgotten), and no record of when personal data was purged from `attrs` |
| `bbl_mutations` | The audit trail is an invariant everywhere, but GDPR right-to-erasure and right-to-be-forgotten may legally oblige purging change history rows for a specific entity too — `diff` can contain personal data captured at the time of each mutation |
| Person ↔ project | No direct person–project membership table; PI, co-PI, and researcher roles can only be inferred through works, which loses people who have no publications yet |
| User proxies | PK is `(user_id, proxy_user_id)` — prevents two separate time-bounded proxy arrangements for the same pair (e.g. two distinct leave periods) |
| Grants | No `revoked_at` — cannot distinguish "grant expired naturally" from "grant was explicitly revoked before expiry"; matters for compliance audits |
| Subscriptions | `topic` is untyped free text with no entity binding and no cleanup when a referenced entity is deleted |
| DDL ordering | `bbl_work_sources.candidate_id` FK references `bbl_work_candidates` which is defined later; FK must be deferred to the candidates migration |
| Subscriptions | `bbl_subscriptions` is too thin for webhook delivery: no headers, no HMAC secret, no enabled flag, no retry tracking |

---

## Greenfield schema

```sql
-- ============================================================
-- COLLATION
-- ============================================================

CREATE COLLATION bbl_case_insensitive (
    provider = icu,
    locale = 'und-u-ks-level2',
    deterministic = false
);

-- ============================================================
-- SOURCE REGISTRY
-- Defines known import sources and their trust priority.
-- priority is informational: ingest paths use it to rank sources when populating
-- provenance.field_source; it does not trigger automatic field overwrites.
-- ============================================================

CREATE TABLE bbl_sources (
    id          text PRIMARY KEY,              -- e.g. 'ugent_ldap', 'orcid', 'plato', 'manual'
    label       text NOT NULL,
    priority    int NOT NULL DEFAULT 0,        -- higher = preferred; informational ranking for provenance.field_source; does not trigger automatic field overwrites
    description text
);

-- ============================================================
-- USERS
-- Application accounts.
-- person_identity_id is the explicit link to the canonical person authority record.
-- It is nullable (service/admin accounts have no person identity) and unique
-- (two accounts cannot claim the same identity).
-- Set and removed through user management commands; audited in bbl_user_events.
-- Forward FK added via ALTER TABLE after bbl_person_identities is defined below.
-- ============================================================

CREATE TABLE bbl_users (
    id                 uuid PRIMARY KEY,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
    username           text NOT NULL UNIQUE,
    email              text NOT NULL COLLATE bbl_case_insensitive,
    name               text NOT NULL,
    role               text NOT NULL,               -- 'admin' | 'curator' | 'user'
    deactivate_at      timestamptz,
    person_identity_id uuid UNIQUE  -- FK added below: REFERENCES bbl_person_identities (id) ON DELETE SET NULL
);

-- Lightweight security audit log for user account events.
-- Separate from bbl_mutations: user mutations are access-control history,
-- not domain history; different retention and access characteristics.
-- kind: 'role_changed' | 'deactivated' | 'reactivated' |
--       'identity_linked' | 'identity_unlinked' | 'proxy_granted' | 'proxy_revoked'
CREATE TABLE bbl_user_events (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    actor_id   uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    payload    jsonb NOT NULL DEFAULT '{}',  -- e.g. old/new role, identity id, proxy target
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE INDEX ON bbl_user_events (user_id);

-- Authentication identifiers only: SSO claims (e.g. OIDC sub, ugent_id from LDAP).
-- Used to match incoming login tokens to a bbl_users row.
-- Not used as signals for person authority matching — that is bbl_person_record_identifiers.
CREATE TABLE bbl_user_identifiers (
    user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    scheme  text NOT NULL,
    val     text NOT NULL,
    PRIMARY KEY (user_id, scheme),
    UNIQUE (scheme, val)
);

-- Source provenance for users. One row per (user, source).
-- last_seen_at is stamped by the ingest layer on every harvest sweep.
-- expires_at NULL = permanent (one-time imports, manually added users);
-- recurring sources set expires_at so Catbird can detect absent users.
-- Catbird staleness sweep: last_seen_at < sweep_started_at AND expires_at IS NOT NULL
-- → set bbl_users.deactivate_at.
CREATE TABLE bbl_user_sources (
    user_id          uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    last_seen_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at       timestamptz,   -- NULL = permanent; set by recurring sources
    PRIMARY KEY (user_id, source)
);

CREATE INDEX ON bbl_user_sources (source, last_seen_at) WHERE expires_at IS NOT NULL;

-- Auth methods: which named auth provider a user authenticates through.
-- provider references a registered AuthProvider by its ID (e.g. 'ugent_oidc',
-- 'orcid_oidc', 'magic_link') — not a generic protocol name.
-- Using named providers allows a user to have both 'ugent_oidc' and 'orcid_oidc'.
-- identifier is the provider-specific handle (e.g. OIDC sub, email).
-- Auto-associated by the ingest layer when a UserSource harvests a user;
-- can also be set manually by an admin.
CREATE TABLE bbl_user_auth_methods (
    user_id    uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    provider   text NOT NULL,   -- registered AuthProvider ID
    identifier text NOT NULL,   -- provider-specific handle
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (user_id, provider),
    UNIQUE (provider, identifier)
);

CREATE INDEX ON bbl_user_auth_methods (provider, identifier);

-- Proxy access: user A can act on behalf of user B, within a time window.
-- Surrogate PK allows multiple distinct time-bounded arrangements for the same pair
-- (e.g. two separate leave periods). UNIQUE on (user_id, proxy_user_id, valid_from)
-- prevents exact duplicates while permitting renewals.
CREATE TABLE bbl_user_proxies (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    proxy_user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    valid_from    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    valid_to      timestamptz,
    granted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    UNIQUE (user_id, proxy_user_id, valid_from),
    CHECK (user_id <> proxy_user_id)
);

CREATE INDEX ON bbl_user_proxies (user_id);
CREATE INDEX ON bbl_user_proxies (proxy_user_id);

-- ============================================================
-- GRANTS
-- Single table covering all permission grants for all users.
-- One query gives a complete picture of what a user may do:
--   SELECT * FROM bbl_grants
--   WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > now())
--
-- scope_type / scope_id control what the grant covers:
--   NULL / NULL         global (same weight as bbl_users.role)
--   'organization'/{id} all works and people linked to that org identity
--                       and its descendants (tree walk via bbl_organization_rels)
--   'project'/{id}      all works linked to that project
--   'work'/{id}         a specific work (replaces bbl_work_permissions)
--   'person_identity'/{id}  a specific person identity
--
-- kind values span both role-level and entity-level semantics:
--   role-level:   'admin' | 'curator' | 'submitter' | 'reviewer'
--   entity-level: 'owner' | 'edit' | 'read'
--
-- note is a free-text reason, useful for temporary/ad-hoc grants.
-- ============================================================

CREATE TABLE bbl_grants (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    kind          text NOT NULL,
    scope_type    text,
    scope_id      uuid,
    granted_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    granted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    expires_at    timestamptz,
    revoked_at    timestamptz,   -- set when explicitly revoked before natural expiry; NULL = not revoked
    note          text,
    CHECK ((scope_type IS NULL) = (scope_id IS NULL))
);

CREATE INDEX ON bbl_grants (user_id);
CREATE INDEX ON bbl_grants (scope_type, scope_id);
CREATE INDEX ON bbl_grants (user_id) WHERE revoked_at IS NULL AND expires_at IS NULL;  -- active permanent grants
CREATE INDEX ON bbl_grants (expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================
-- ORGANIZATIONS
-- Single curated table. Orgs come from one source at a time (unlike people);
-- no records+identities split needed.
-- Names and renames are stored in attrs.names as a multilingual array:
--   [{"lang": "nl", "name": "..."}, {"lang": "en", "name": "..."}]
-- lang follows BCP 47; null = language-neutral (e.g. abbreviations).
-- A name change that represents a new org unit gets a new row + successor_of rel.
-- ============================================================

CREATE TABLE bbl_organizations (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind          text NOT NULL,   -- 'faculty' | 'department' | 'research_group' | ...
    attrs               jsonb NOT NULL DEFAULT '{}',
    provenance          jsonb NOT NULL DEFAULT '{}',   -- {field: source} informational; set by ingest path
    attrs_locked_fields text[] NOT NULL DEFAULT '{}'   -- fields a curator has locked; ingest skips these
);

-- Time-bounded identifiers; valid_from/valid_to handle reuse across splits/mergers.
CREATE TABLE bbl_organization_identifiers (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    scheme          text NOT NULL,
    value           text NOT NULL,
    valid_from      timestamptz,
    valid_to        timestamptz,
    revoked_at      timestamptz
);

CREATE INDEX ON bbl_organization_identifiers (organization_id);
CREATE INDEX ON bbl_organization_identifiers (scheme, value);

-- Temporal hierarchical relationships.
-- Supports parent/child, mergers, splits, and successor links over time.
-- kind: 'part_of' | 'merged_into' | 'split_from' | 'successor_of'
CREATE TABLE bbl_organization_rels (
    id                  uuid PRIMARY KEY,
    organization_id     uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    rel_organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    kind                text NOT NULL,
    valid_from          timestamptz,
    valid_to            timestamptz,
    CHECK (organization_id <> rel_organization_id)
);

CREATE INDEX ON bbl_organization_rels (organization_id);
CREATE INDEX ON bbl_organization_rels (rel_organization_id);
-- Current active relationships:
CREATE INDEX ON bbl_organization_rels (organization_id) WHERE valid_to IS NULL;

CREATE TABLE bbl_organization_sources (
    organization_id  uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    ingested_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (organization_id, source)
);

CREATE INDEX ON bbl_organization_sources (source, source_record_id);

-- ============================================================
-- PEOPLE
-- Approach: MDM consolidation with durable source records.
--
-- person_records  = immutable-ish source avatars (one per import payload)
-- person_identities = canonical golden records (one per real-world person)
-- person_identity_records = the consolidation link; carries process metadata
--
-- [source A]──┐
-- [source B]──┼──► person_records ──► person_identity_records ──► person_identities
-- [manual  ]──┘                            (link process)            (golden record)
--
-- A record belongs to at most one active identity.
-- Enforced in command logic, not a hard DB constraint (policy not yet stable).
-- All mutations go through AddRev; the audit trail is in bbl_mutations.
-- ============================================================

CREATE TABLE bbl_person_identities (
    id                  uuid PRIMARY KEY,
    version             int NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs      jsonb NOT NULL DEFAULT '{}',
    provenance jsonb NOT NULL DEFAULT '{}'
);

-- Resolve the forward reference declared on bbl_users above.
ALTER TABLE bbl_users
    ADD CONSTRAINT bbl_users_person_identity_id_fkey
    FOREIGN KEY (person_identity_id)
    REFERENCES bbl_person_identities (id)
    ON DELETE SET NULL;

CREATE TABLE bbl_person_records (
    id               uuid PRIMARY KEY,
    version          int NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at       timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    attrs            jsonb NOT NULL DEFAULT '{}',
    UNIQUE (source, source_record_id)
);

CREATE TABLE bbl_person_identity_records (
    identity_id        uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    record_id          uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    PRIMARY KEY (identity_id, record_id),
    status             text NOT NULL DEFAULT 'active',
    link_kind          text NOT NULL,
    confidence         numeric,
    decided_by_user_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE INDEX ON bbl_person_identity_records (record_id);
CREATE INDEX ON bbl_person_identity_records (status);

CREATE TABLE bbl_person_identifiers (
    id         uuid PRIMARY KEY,
    scheme     text NOT NULL,
    value text NOT NULL,
    UNIQUE (scheme, value)
);

CREATE INDEX ON bbl_person_identifiers (scheme, value);

CREATE TABLE bbl_person_record_identifiers (
    record_id     uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_person_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (record_id, identifier_id)
);

CREATE INDEX ON bbl_person_record_identifiers (identifier_id);

-- Temporal affiliations between person identities and organization identities.
-- Replaces the flat bbl_person_organizations table.
CREATE TABLE bbl_person_affiliations (
    id              uuid PRIMARY KEY,
    person_id       uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    role            text,                   -- e.g. 'researcher', 'professor', 'phd_student'
    valid_from      timestamptz,
    valid_to        timestamptz,
    source          text REFERENCES bbl_sources (id)
);

CREATE INDEX ON bbl_person_affiliations (person_id);
CREATE INDEX ON bbl_person_affiliations (organization_id);
CREATE INDEX ON bbl_person_affiliations (person_id) WHERE valid_to IS NULL;

-- Match candidate queue (for both people and orgs, typed separately).
CREATE TABLE bbl_person_match_candidates (
    id                 uuid PRIMARY KEY,
    record_id_a        uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    record_id_b        uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    status             text NOT NULL DEFAULT 'open',  -- open | accepted | rejected
    confidence         numeric NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
    decided_by_user_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    CHECK (record_id_a <> record_id_b)
);

CREATE INDEX ON bbl_person_match_candidates (status);
CREATE INDEX ON bbl_person_match_candidates (record_id_a);
CREATE INDEX ON bbl_person_match_candidates (record_id_b);

CREATE TABLE bbl_person_match_scores (
    candidate_id uuid NOT NULL REFERENCES bbl_person_match_candidates (id) ON DELETE CASCADE,
    signal       text NOT NULL,     -- e.g. 'orcid_exact', 'name_fuzzy', 'ugent_id_exact'
    score        numeric NOT NULL,  -- 0.0 – 1.0
    weight       numeric NOT NULL,
    matched_a    text,
    matched_b    text,
    PRIMARY KEY (candidate_id, signal)
);

-- ============================================================
-- PROJECTS
-- ============================================================

CREATE TABLE bbl_projects (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    status        text NOT NULL DEFAULT 'active',
    starts_on           date,
    ends_on             date,
    attrs               jsonb NOT NULL DEFAULT '{}',
    provenance          jsonb NOT NULL DEFAULT '{}',   -- {field: source} informational; set by ingest path
    attrs_locked_fields text[] NOT NULL DEFAULT '{}'   -- fields a curator has locked; ingest skips these
);

CREATE TABLE bbl_project_identifiers (
    id         uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    scheme     text NOT NULL,
    value      text NOT NULL,
    UNIQUE (project_id, scheme, value)
);

CREATE INDEX ON bbl_project_identifiers (project_id);
CREATE INDEX ON bbl_project_identifiers (scheme, value);

CREATE TABLE bbl_project_sources (
    project_id       uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    ingested_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (project_id, source)
);

CREATE INDEX ON bbl_project_sources (source, source_record_id);

-- Person ↔ project roles.
-- A work-centric join (bbl_work_projects + bbl_work_contributors) cannot express PIs
-- or co-investigators who have no publications on the project yet.
-- role: 'pi' | 'co_pi' | 'researcher' | 'phd_student' | ...
CREATE TABLE bbl_person_project_roles (
    id             uuid PRIMARY KEY,
    person_id      uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    project_id     uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    role           text,
    valid_from     date,
    valid_to       date,
    source         text REFERENCES bbl_sources (id),
    UNIQUE (person_id, project_id, role, valid_from)  -- valid_from in key allows same role after a gap
);

CREATE INDEX ON bbl_person_project_roles (person_id);
CREATE INDEX ON bbl_person_project_roles (project_id);
CREATE INDEX ON bbl_person_project_roles (person_id) WHERE valid_to IS NULL;

-- ============================================================
-- WORKS
-- ============================================================

-- The row is NEVER hard-deleted once a work has been public. External citations, DOIs,
-- OAI-PMH tombstones, and persistent URLs require the id to remain resolvable indefinitely.
-- Draft works (status='draft', never transitioned to 'public') may be hard-deleted freely;
-- their bbl_mutations rows can be deleted in the same transaction with no special ceremony.
--
-- For legally-mandated takedowns (GDPR, patent, right to be forgotten) the row
-- stays but attrs is purged (set to '{}'). Whether attrs, changes history, or both
-- are purged is tracked in bbl_work_takedowns (attrs_purged_at, mutations_purged_at).
-- delete_kind distinguishes routine editorial events from legal obligations:
--   'withdrawn'  = author/editor request post-publication; tombstone only, no purge
--   'retracted'  = post-publication integrity issue; tombstone only, no purge
--   'takedown'   = legal obligation; bbl_work_takedowns row required; attrs and/or
--                  changes history purged depending on legal_basis
CREATE TABLE bbl_works (
    id             uuid PRIMARY KEY,
    version        int NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind           text NOT NULL,     -- 'journal_article' | 'book' | 'dataset' | ...
    status         text NOT NULL,     -- 'draft' | 'submitted' | 'public' | 'deleted'
    delete_kind    text,              -- 'withdrawn' | 'retracted' | 'takedown'; set with status='deleted'
    deleted_at     timestamptz,       -- set when status transitions to 'deleted'
    deleted_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs               jsonb NOT NULL DEFAULT '{}',  -- '{}' when attrs have been purged via takedown
    provenance          jsonb NOT NULL DEFAULT '{}',  -- {field: source} informational; set by ingest path
    attrs_locked_fields text[] NOT NULL DEFAULT '{}'  -- fields a curator has locked; ingest skips these
);

CREATE INDEX ON bbl_works (status);

-- Legal takedown record. One row per takedown decision.
-- Survives after purging; subject to its own data retention policy.
-- legal_basis: 'gdpr_erasure' | 'right_to_be_forgotten' | 'patent' | 'court_order' | 'other'
-- attrs_purged_at / mutations_purged_at: NULL = not yet purged; set when each purge is executed.
-- Both may be set, either, or neither depending on legal_basis and policy.
CREATE TABLE bbl_work_takedowns (
    id                uuid PRIMARY KEY,
    work_id           uuid NOT NULL REFERENCES bbl_works (id),  -- intentionally no CASCADE
    legal_basis       text NOT NULL,
    reference         text,            -- external case/ticket reference
    requested_at      timestamptz NOT NULL,
    requested_by      text,            -- name/org of requesting party; free text
    decided_at        timestamptz,
    decided_by_id     uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs_purged_at   timestamptz,     -- when bbl_works.attrs was set to '{}'
    mutations_purged_at timestamptz,     -- when bbl_mutations rows for this work were deleted
    notes             text
);

CREATE INDEX ON bbl_work_takedowns (work_id);

-- Source provenance: which external systems contributed to this work.
-- One row per (work, source). Multiple rows when the same work is ingested
-- from WoS, ORCID, and a manual entry independently.
-- candidate_id links back to the staging row that triggered creation or last merge;
-- NULL for manually created works.
-- ingested_at tracks the most recent import from this source.
CREATE TABLE bbl_work_sources (
    work_id          uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    candidate_id     uuid,   -- soft ref to bbl_work_candidates; FK added via ALTER TABLE in the candidates migration (forward reference)
    ingested_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ingested_rev_id  uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    PRIMARY KEY (work_id, source)
);

CREATE INDEX ON bbl_work_sources (source, source_record_id);
CREATE INDEX ON bbl_work_sources (candidate_id) WHERE candidate_id IS NOT NULL;

CREATE TABLE bbl_work_identifiers (
    id      uuid PRIMARY KEY,
    work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme  text NOT NULL,
    value   text NOT NULL,
    source  text REFERENCES bbl_sources (id),
    UNIQUE (work_id, scheme, value)
);

CREATE INDEX ON bbl_work_identifiers (work_id);
CREATE INDEX ON bbl_work_identifiers (scheme, value);

-- Contributors: person_identity_id links to canonical identity.
-- person_identity_snapshot preserves name/role as recorded at time of contribution
-- so contributor attribution survives renames and dedup merges.
-- person_identity_id nullable: unmatched contributors remain without a link.
CREATE TABLE bbl_work_contributors (
    work_id                  uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    pos                      text NOT NULL COLLATE "C",  -- fracdex; author order is semantically meaningful
    person_identity_id       uuid REFERENCES bbl_person_identities (id) ON DELETE SET NULL,
    person_identity_snapshot jsonb NOT NULL DEFAULT '{}',  -- name, role etc at time of entry
    role                     text,                          -- 'author' | 'editor' | 'translator' | ...
    attrs                    jsonb NOT NULL DEFAULT '{}',  -- extra source-specific fields not covered by snapshot/role
    PRIMARY KEY (work_id, pos)
);

CREATE INDEX ON bbl_work_contributors (person_identity_id) WHERE person_identity_id IS NOT NULL;

-- Work ↔ organization links (affiliation at time of work, not temporal).
CREATE TABLE bbl_work_organizations (
    work_id         uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq             int NOT NULL,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE RESTRICT,
    role            text,
    PRIMARY KEY (work_id, seq),
    UNIQUE (work_id, organization_id)
);

CREATE INDEX ON bbl_work_organizations (organization_id);

-- Work ↔ project links.
CREATE TABLE bbl_work_projects (
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq        int NOT NULL,
    project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE RESTRICT,
    PRIMARY KEY (work_id, seq),
    UNIQUE (work_id, project_id)
);

CREATE INDEX ON bbl_work_projects (project_id);

-- Work-to-work relationships (e.g. is_part_of, has_translation, has_correction).
-- Directional: (work_id) --kind--> (rel_work_id).
-- Asymmetric kinds (is_part_of, has_correction): one row, direction is meaningful.
-- Symmetric kinds (has_translation, is_duplicate_of): store one row, query both sides.
-- Querying all works related to X always requires:
--   WHERE work_id = X OR rel_work_id = X
-- Both sides are indexed; the partial index on rel_work_id covers reverse lookups.
CREATE TABLE bbl_work_rels (
    work_id      uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq          int NOT NULL,
    kind         text NOT NULL,
    rel_work_id  uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    PRIMARY KEY (work_id, seq),
    CHECK (work_id <> rel_work_id)
);

CREATE INDEX ON bbl_work_rels (rel_work_id);

-- Files: includes checksum, upload status, and access control.
CREATE TABLE bbl_work_files (
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq           int NOT NULL,
    object_id     uuid NOT NULL,
    name          text NOT NULL,
    content_type  text NOT NULL,
    size          int NOT NULL,
    sha256        text,           -- populated after upload confirmation
    upload_status text NOT NULL DEFAULT 'pending',  -- pending | complete | failed
    access_kind         text NOT NULL DEFAULT 'open',  -- open | restricted | closed
    embargo_until       timestamptz,     -- planned lift date; preserved after lift as bibliographic record
    embargo_access_kind text,            -- access kind after lift; preserved after lift
    embargo_lifted_at   timestamptz,     -- NULL = embargo active (or never embargoed); set when Catbird applies the transition
    PRIMARY KEY (work_id, seq)
);

-- Review message thread for a work. Only populated when there is back-and-forth
-- between curator and submitter; a direct PublishWork leaves no row here.
-- Submission history (who submitted when) is in bbl_revs/bbl_mutations.
-- Catbird is used only to dispatch notifications as a side effect of commands.
-- seq is append-only (1, 2, 3 …); messages are never reordered.
-- kind: 'submitted'      = cover note from submitter (optional)
--       'review_comment' = curator or submitter comment
--       'returned'       = curator returned to draft; body is reason
--       'published'      = curator note on publish (optional)
CREATE TABLE bbl_work_review_messages (
    id         uuid PRIMARY KEY,
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq        int  NOT NULL,
    rev_id     uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    author_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind       text NOT NULL,
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (work_id, seq)
);
CREATE INDEX ON bbl_work_review_messages (work_id);

-- Work-level permissions (bbl_work_permissions in the current schema) are subsumed
-- by bbl_grants with scope_type='work', scope_id=<work_id>.
-- Migration: INSERT INTO bbl_grants (user_id, kind, scope_type, scope_id, granted_at, expires_at)
--            SELECT user_id, kind, 'work', work_id, granted_at, expires_at FROM bbl_work_permissions;

-- ============================================================
-- REPRESENTATIONS & SETS
-- Precomputed serialization cache for works (OAI-PMH, CSL, MODS, ...).
-- entity_version is the work.version at render time; stale when work.version > this.
-- record_sha256 detects no-op re-renders: UpsertWorkRepresentation skips the write
-- and does not advance updated_at when the hash matches.
-- ============================================================

CREATE TABLE bbl_work_collections (
    id          uuid PRIMARY KEY,
    name        text NOT NULL UNIQUE,
    description text
);

CREATE TABLE bbl_work_representations (
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme         text NOT NULL,    -- 'oai_dc' | 'mods' | 'csl' | ...
    record         bytea NOT NULL,
    record_sha256  bytea NOT NULL,
    work_version   int NOT NULL,
    updated_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (work_id, scheme)
);

CREATE INDEX ON bbl_work_representations (updated_at);

-- Named work groups (OAI-PMH sets, open access subsets, faculty feeds, ...).
CREATE TABLE bbl_work_collection_works (
    collection_id uuid NOT NULL REFERENCES bbl_work_collections (id) ON DELETE CASCADE,
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    pos           text NOT NULL COLLATE "C",  -- fracdex; display order within collection
    PRIMARY KEY (collection_id, work_id),
    UNIQUE (collection_id, pos)
);

CREATE INDEX ON bbl_work_collection_works (work_id);

-- ============================================================
-- AUDIT: REVS + CHANGES
-- ============================================================

-- Rev now carries an optional source context for system-initiated changes.
CREATE TABLE bbl_revs (
    id         uuid PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    user_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    source     text REFERENCES bbl_sources (id)  -- set for automated import revs
);

-- Mutations: one row per named mutation applied in a rev.
-- name matches the registered MutationImpl ('SetTitle', 'PublishWork', ...).
-- Explicit entity_type discriminator + op_type; no CHECK arithmetic.
--
-- Legal exception: bbl_mutations rows for a specific entity_id MAY be hard-deleted in
-- two sanctioned cases:
--   1. The work is still a draft (status='draft', never public): hard-delete the work
--      row and its mutations in the same transaction; no special tracking needed.
--   2. A GDPR erasure or right-to-be-forgotten takedown is actioned on a public work:
--      diff can contain personal data captured at mutation time. The decision and
--      timestamp are recorded in bbl_work_takedowns.mutations_purged_at before rows
--      are removed.
-- These are the only sanctioned hard-delete paths in the schema.
CREATE TABLE bbl_mutations (
    id          bigserial PRIMARY KEY,
    rev_id      uuid NOT NULL REFERENCES bbl_revs (id),  -- no cascade: revs are immutable
    name        text NOT NULL,       -- registered MutationImpl name
    entity_type text NOT NULL,       -- 'organization' | 'person_identity' | 'person_record' | 'project' | 'work'
    entity_id   uuid NOT NULL,
    op_type     text NOT NULL,       -- 'create' | 'update' | 'delete'
    diff        jsonb NOT NULL       -- {args: {...}, prev: {...}}
);

CREATE INDEX ON bbl_mutations (rev_id);
CREATE INDEX ON bbl_mutations (entity_type, entity_id);

-- ============================================================
-- LISTS & SUBSCRIPTIONS
-- Lists are persistent named collections of entities.
-- entity_type on bbl_lists optionally constrains all items to one type
-- (e.g. a reading list of works only). NULL = heterogeneous list.
-- Homogeneity is enforced in command logic, not a DB constraint,
-- because a CHECK on bbl_list_items can't reference bbl_lists.entity_type.
-- ============================================================

CREATE TABLE bbl_lists (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    name          text NOT NULL,
    public        boolean NOT NULL DEFAULT false,
    entity_type   text,                -- NULL = heterogeneous; set to lock list to one type
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL
);

CREATE INDEX ON bbl_lists (created_by_id);

CREATE TABLE bbl_list_items (
    list_id     uuid NOT NULL REFERENCES bbl_lists (id) ON DELETE CASCADE,
    entity_type text NOT NULL,   -- 'work' | 'person_identity' | 'organization' | ...
    entity_id   uuid NOT NULL,
    pos         text NOT NULL COLLATE "C",
    UNIQUE (list_id, entity_type, entity_id),
    UNIQUE (list_id, pos)
);

CREATE INDEX ON bbl_list_items (entity_type, entity_id);

-- topic is a structured event kind e.g. 'work.updated', 'person_identity.merged'.
--
-- Webhook delivery:
--   webhook_url NULL  = internal notification only (centrifugo / catbird); other webhook columns ignored.
--   webhook_secret    = used to produce an HMAC-SHA256 signature sent as X-Webhook-Signature on every
--                       delivery. Stored encrypted at rest. NULL = unsigned deliveries.
--   webhook_headers   = extra request headers as {"Header-Name": "value"} — use for Authorization:
--                       Bearer <token>, or any integration-specific header. Values stored encrypted.
--
-- Reliability:
--   status            = 'active' | 'disabled' (user) | 'suspended' (auto on excess failures)
--   failure_count     = consecutive delivery failures; reset to 0 on any successful delivery.
--   last_attempted_at / last_succeeded_at = updated by the catbird dispatcher after each job.
--
-- Delivery is handled by catbird: each event enqueues a catbird job carrying the
-- subscription_id + serialized payload. Per-attempt history (status, http_status,
-- duration, error, retries) lives in catbird's own job tables; no separate delivery
-- log table is needed here.
CREATE TABLE bbl_subscriptions (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    topic               text NOT NULL,
    webhook_url         text,                              -- NULL = internal only
    webhook_secret      text,                              -- HMAC-SHA256 signing secret; encrypted at rest
    webhook_headers     jsonb NOT NULL DEFAULT '{}',       -- extra headers; values encrypted at rest
    status              text NOT NULL DEFAULT 'active',    -- active | disabled | suspended
    failure_count       int NOT NULL DEFAULT 0,
    last_attempted_at   timestamptz,
    last_succeeded_at   timestamptz,
    created_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CHECK (webhook_url IS NOT NULL OR (webhook_secret IS NULL AND webhook_headers = '{}'))
);

CREATE INDEX ON bbl_subscriptions (user_id);
CREATE INDEX ON bbl_subscriptions (topic) WHERE status = 'active';  -- delivery dispatcher

-- Delivery log, retry scheduling, and per-attempt history are handled by catbird
-- (PostgreSQL-backed job queue). Each event enqueues one catbird job per matching
-- subscription; catbird owns status, attempt count, next_attempt_at, and error recording.
-- The subscription row is updated (failure_count, suspended_at, last_attempted_at,
-- last_succeeded_at) by the catbird job handler on completion.
```

---

## Person identity — command surface (AddRev operations)

All mutations go through `AddRev`; one rev = one transaction.

| Command | Description |
|---|---|
| `IngestPersonRecord(source, source_record_id, attrs)` | Import or refresh a source record |
| `CreatePersonIdentity()` | Create a new canonical identity (triggered by first accepted link) |
| `LinkRecordToIdentity(record_id, identity_id, link_kind, confidence)` | Attach a record to an identity (manual or auto) |
| `UnlinkRecordFromIdentity(record_id, identity_id)` | Detach a record; identity remains |
| `AcceptMatchCandidate(candidate_id)` | Accept a proposed link; triggers `LinkRecordToIdentity` |
| `RejectMatchCandidate(candidate_id)` | Dismiss a proposed link |
| `ResolveIdentityProfile(identity_id)` | Recompute `attrs` + `provenance` from active records |

---

## Repository — method surface

A single `Repository` backed by one PostgreSQL connection pool. All commands go through
`AddRev(ctx, userID, source, func(rev) error) (revID, error)` — one transaction, one
`bbl_revs` row, one or more `bbl_mutations` rows. Queries are plain reads, no rev needed.

A split into multiple repos is not warranted: `AddRev` is a shared primitive, queries
join across entity boundaries, and `bbl_grants`/`bbl_mutations` are genuinely cross-entity.

#### Identifier-based addressing

Commands that reference related entities (e.g. `AddWorkProject`, `MergeAttrs` on a
work ingested by a harvester) require an internal UUID. External callers — harvesters,
importers, API clients — often know only a domain identifier: a project grant number,
an ORCID, a DOI, a WoS ID.

The resolution model keeps the core command layer UUID-only:

```
Ref = uuid | {scheme: text, value: text}
```

A `ResolveRef(entityType, ref) → uuid` helper runs inside the same transaction before
the command executes. If the identifier maps to no entity, the caller may choose to
create one first (`IngestPersonRecord`, `CreateProject`, …) or treat the absence as a
validation error. The resolved UUID is what gets stored in `bbl_mutations.diff` — the
audit trail is always in terms of internal IDs.

Examples:
```
AddWorkProject(workID, Ref{scheme: "fwo_grant", value: "G001234N"})
  → resolves to project uuid, then proceeds as AddWorkProject(workID, <uuid>)

MergeAttrs(Ref{scheme: "doi", value: "10.1000/xyz"}, attrs)
  → resolves to work uuid, then proceeds as MergeAttrs(<uuid>, attrs)
```

`ResolveRef` is a thin lookup against the relevant `bbl_*_identifiers` table. It does not create entities.

### Users

| Method | Type | Description |
|---|---|---|
| `GetUser(id)` | query | Fetch by primary key |
| `GetUserByUsername(username)` | query | Login lookup |
| `GetUserByAuthMethod(provider, identifier)` | query | Match incoming auth claim to a user |
| `ListUsers(opts)` | query | Paginated list with filters (role, deactivated, search) |
| `ListStaleUserSources(source, sweepStartedAt)` | query | Users from source with `last_seen_at < sweepStartedAt AND expires_at IS NOT NULL`; drives Catbird deactivation sweep |
| `CreateUser(attrs)` | command | New application account |
| `UpdateUser(id, attrs)` | command | Profile update |
| `DeactivateUser(id)` | command | Set `deactivate_at`; does not hard-delete |
| `LinkUserToIdentity(userID, identityID)` | command | Set `bbl_users.person_identity_id`; clears previous link |
| `UnlinkUserFromIdentity(userID)` | command | Null out `person_identity_id` |
| `UpsertUserSource(userID, source, sourceRecordID, expiresAt)` | command | Stamp `last_seen_at`; insert on first sight |
| `AddUserAuthMethod(userID, provider, identifier)` | command | Associate a named auth provider with a user |
| `RemoveUserAuthMethod(userID, provider)` | command | Deassociate a provider |
| `SetUserProxy(userID, proxyUserID, validFrom, validTo)` | command | Grant full proxy delegation |
| `RemoveUserProxy(id)` | command | Remove a proxy row by surrogate PK |

### Organizations

| Method | Type | Description |
|---|---|---|
| `GetOrganization(id)` | query | Fetch org with identifiers |
| `ListOrganizations(opts)` | query | Paginated list with filters (kind, search) |
| `GetOrganizationTree(id, at)` | query | Ancestor/descendant walk via `bbl_organization_rels` at a point in time |
| `CreateOrganization(kind, attrs)` | command | New org |
| `UpdateOrganization(id, attrs)` | command | Update attrs (including names) |
| `AddOrganizationIdentifier(orgID, scheme, value, validFrom, validTo)` | command | Add a time-bounded identifier |
| `RevokeOrganizationIdentifier(id)` | command | Set `revoked_at` |
| `AddOrganizationRel(orgID, relOrgID, kind, validFrom, validTo)` | command | Add temporal hierarchy or merger fact |
| `RemoveOrganizationRel(id)` | command | End or delete a relationship |

### People

| Method | Type | Description |
|---|---|---|
| `GetPersonIdentity(id)` | query | Fetch canonical identity with resolved attrs |
| `GetPersonRecord(id)` | query | Fetch a source record |
| `ListPersonIdentities(opts)` | query | Paginated list with filters (org, search) |
| `ListPersonMatchCandidates(opts)` | query | Backoffice review queue; filterable by status, confidence |
| `ListPersonAffiliations(personID, at)` | query | Active or all affiliations at a point in time |
| `IngestPersonRecord(source, sourceRecordID, attrs)` | command | Import or refresh a source record |
| `CreatePersonIdentity()` | command | New canonical identity |
| `LinkPersonRecordToIdentity(recordID, identityID, linkKind, confidence)` | command | Attach record to identity |
| `UnlinkPersonRecordFromIdentity(recordID, identityID)` | command | Detach record; identity remains |
| `AcceptPersonMatchCandidate(candidateID)` | command | Accept proposed link; triggers `LinkPersonRecordToIdentity` |
| `RejectPersonMatchCandidate(candidateID)` | command | Dismiss proposed link |
| `ResolvePersonProfile(identityID)` | command | Recompute `attrs` + `provenance` from active records |
| `AddPersonAffiliation(personID, orgID, role, validFrom, validTo, source)` | command | Record org affiliation |
| `RemovePersonAffiliation(id)` | command | End or delete an affiliation |
| `AddPersonProjectRole(personID, projectID, role, validFrom, validTo)` | command | Record project membership |
| `RemovePersonProjectRole(id)` | command | End or delete a project role |

### Projects

| Method | Type | Description |
|---|---|---|
| `GetProject(id)` | query | Fetch by primary key |
| `ListProjects(opts)` | query | Paginated list with filters (status, search) |
| `ListProjectMembers(projectID)` | query | All `bbl_person_project_roles` for a project |
| `CreateProject(attrs)` | command | New project |
| `UpdateProject(id, attrs)` | command | Update attrs and/or status/dates |

### Works

| Method | Type | Description |
|---|---|---|
| `GetWork(id)` | query | Fetch work with contributors, files, orgs, projects |
| `ListWorks(opts)` | query | Paginated/cursor list with filters (kind, status, person, org, project) |
| `GetWorkHistory(id)` | query | `bbl_mutations` rows for a work ordered by rev |
| `CreateWork(kind, attrs)` | command | New draft work; inserts `bbl_grants` owner row for creating user |
| `UpdateWork(id, attrs)` | command | Update metadata |
| `SubmitWork(id)` | command | Transition `draft → submitted`; appends `kind='submitted'` message to the work's review thread; no Catbird flow |
| `PublishWork(id)` | command | Transition `submitted → public`; no message required |
| `ReturnToDraft(id, message)` | command | Transition `submitted → draft`; appends `kind='returned'` message with curator reason; Catbird notifies submitter |
| `WithdrawWork(id)` | command | Transition `public → deleted`, `delete_kind='withdrawn'` |
| `RetractWork(id)` | command | Transition `public → deleted`, `delete_kind='retracted'` |
| `DeleteDraftWork(id)` | command | Hard-delete; only valid while `status='draft'`; deletes `bbl_mutations` rows in same transaction |
| `AddWorkContributor(workID, pos, attrs)` | command | Add contributor row; resolves identity link if possible |
| `UpdateWorkContributor(workID, pos, attrs)` | command | Update contributor attrs or identity link |
| `RemoveWorkContributor(workID, pos)` | command | Remove contributor by position |
| `AddWorkReviewComment(id, message)` | command | Curator or submitter appends `kind='review_comment'` message to the open workflow; Catbird notifies the other party |
| `AddWorkSource(workID, source, sourceRecordID, candidateID)` | command | Record import provenance |
| `UpdateWorkSource(workID, source, sourceRecordID)` | command | Update source record ID after re-sync |

### Takedowns

| Method | Type | Description |
|---|---|---|
| `GetWorkTakedown(id)` | query | Fetch takedown record |
| `ListWorkTakedowns(opts)` | query | Paginated list filterable by status, legal_basis |
| `RequestTakedown(workID, legalBasis, reference, requestedAt, requestedBy)` | command | Open a takedown request |
| `DecideTakedown(id, decided)` | command | Accept or reject the request |
| `PurgeWorkAttrs(takedownID)` | command | Set `bbl_works.attrs = '{}'`, record `attrs_purged_at`; only after decision |
| `PurgeWorkMutations(takedownID)` | command | Delete `bbl_mutations` rows for the work, record `mutations_purged_at`; only for gdpr/rtbf legal bases |

### Candidates

| Method | Type | Description |
|---|---|---|
| `GetWorkCandidate(id)` | query | Fetch with extracted identifiers and person/org suggestions |
| `ListWorkCandidates(opts)` | query | Backoffice review queue; filterable by status, source, person, org, confidence |
| `IngestWorkCandidate(source, sourceRecordID, attrs)` | command | Insert or refresh candidate; extracts identifiers; skips if already accepted/rejected |
| `AcceptWorkCandidate(candidateID)` | command | Create work via `CreateWork`, set `status='accepted'`, insert `bbl_work_sources` row |
| `RejectWorkCandidate(candidateID)` | command | Set `status='rejected'`, clear `attrs` |

### Grants

| Method | Type | Description |
|---|---|---|
| `ListGrantsForUser(userID)` | query | All active grants for a user (one-query access overview) |
| `ListGrantsForScope(scopeType, scopeID)` | query | All grants covering a specific entity |
| `Grant(userID, kind, scopeType, scopeID, expiresAt, note)` | command | Issue a new grant |
| `RevokeGrant(id)` | command | Set `revoked_at = now()`; does not delete the row |
| `ExpireGrant(id, expiresAt)` | command | Shorten or set the natural expiry |

### Lists

| Method | Type | Description |
|---|---|---|
| `GetList(id)` | query | Fetch list metadata |
| `ListLists(opts)` | query | Paginated list; filterable by owner, public, entity_type |
| `GetListItems(listID, opts)` | query | Items in fracdex order |
| `CreateList(name, entityType, public)` | command | New named collection |
| `UpdateList(id, attrs)` | command | Rename or toggle public |
| `DeleteList(id)` | command | Hard-delete list and all items |
| `AddListItem(listID, entityType, entityID, pos)` | command | Append or insert at position |
| `MoveListItem(listID, entityType, entityID, newPos)` | command | Reorder by updating fracdex `pos` |
| `RemoveListItem(listID, entityType, entityID)` | command | Delete item row |

### Subscriptions

| Method | Type | Description |
|---|---|---|
| `ListSubscriptions(userID)` | query | All subscriptions for a user |
| `GetSubscriptionsForTopic(topic)` | query | Active subscriptions matching a fired event; used by catbird dispatcher |
| `CreateSubscription(userID, topic, webhookURL, secret, headers)` | command | New subscription |
| `UpdateSubscription(id, attrs)` | command | Change URL, headers, or topic |
| `EnableSubscription(id)` | command | Set `status='active'`, reset `failure_count` |
| `DisableSubscription(id)` | command | Set `status='disabled'` |
| `DeleteSubscription(id)` | command | Hard-delete |
| `RecordSubscriptionDelivery(id, succeeded, httpStatus, err)` | command | Update `failure_count`, `status`, `last_attempted_at`, `last_succeeded_at`; called by catbird job handler |

### Representations

| Method | Type | Description |
|---|---|---|
| `GetRepresentation(entityType, entityID, scheme)` | query | Fetch cached serialized form |
| `ListStaleRepresentations(entityType)` | query | Rows where `entity_version < current entity.version`; drives catbird batch re-render |
| `UpsertRepresentation(entityType, entityID, scheme, record, entityVersion)` | command | Insert or replace cache; computes SHA-256 of `record`, skips write if hash matches stored `record_sha256`; advances `updated_at` only when content changes |
| `DeleteRepresentation(entityType, entityID, scheme)` | command | Invalidate cache entry; next render re-populates it |

---

## Key design decisions and rationale

### Field model and profiles

Bibliographic metadata for works lives in `attrs jsonb`. Go types define the full
universe of possible fields per entity kind — what can exist. Profile configuration
(which fields are active, required, optional, or locked for a given work kind at a
given installation) lives in a config file (YAML/TOML), not in Go struct tags or the
database, so third-party installations can customize profiles without editing source
code.

**Responsibilities:**
- **Go types** — define the complete field surface; the DB accepts `attrs` as opaque
  jsonb and Go enforces its shape
- **Config file** — defines profiles per work kind (active/required/optional/locked
  fields); the customization point for third-party installations
- **Go startup validation** — any field name in the config that is not a known Go
  field is a startup error, preventing silent drift
- **Search engine** — handles queryability of metadata fields; promoting attrs fields
  to SQL columns is only warranted for constraint enforcement or partial-index
  efficiency, not for SQL queryability

**End-user visibility of the field model:**
- *Form generation* — the work edit form renders only the fields active in the loaded
  profile for that work kind; this is the primary user-facing expression of profiles
- *Submission guidelines* — a page per work kind listing expected fields, rendered
  from the loaded profile config; replaces hand-maintained documentation
- *Field reference (backoffice)* — a data dictionary page showing active fields,
  labels, and constraints per work kind; useful for curators and admins
- *API endpoint* — `GET /api/work-kinds/{kind}/profile` returning the active field
  list for external tools and import mappers

Go can emit a reference config listing all available fields and their defaults; this
serves as the field catalog and documentation for operators and third-party installers.

**Model evolution conventions:**

Profile changes are infrequent, consequential, and made by content specialists or
third-party maintainers — not necessarily Go developers. Three independent axes evolve
at different rates and require different handling:

*SQL schema* — structural column changes use standard goose migrations: add nullable
columns first, backfill, then add constraints. Never rename a column in a single step.
The `attrs jsonb` column itself never changes.

*Go model* — field additions are free; old records read missing fields as zero/absent
values (use `Opt[T]` to distinguish missing-in-DB from zero). Field renames and
removals are two-phase: add new name / migrate data / remove old, across separate
releases. Be lenient on read (ignore unknown fields in attrs), strict on write (only
write known fields). A profile deactivation should always precede a Go model removal:
deactivate first (data preserved in attrs), confirm no active reliance, then remove
from Go with a scrub migration.

*Profile edits* — deactivation hides a field from the UI but never deletes its data
from attrs. Purge is an explicit separate operation (`bbl work scrub-field --field=x`).
Making an optional field required is blocked until existing records are clean or an
explicit override is given.

**Profile loading:**

The profile is loaded from the config file at startup and held in memory for the
lifetime of the process. No hot reload — the profile is fixed for the lifetime of a
running process, avoiding mid-session form inconsistency.

There is no separate apply command. Profile changes follow the normal deploy cycle:
edit the config file, run `bbl profile check` (see below), commit, deploy, restart.
Git is the audit trail.

**`bbl profile diff` — read-only preview before deploying a change:**

Before committing and deploying a profile change, operators run `bbl profile diff`
against the live DB to understand data impact. Read-only; produces no side effects.

```
bbl profile diff profile.yaml

  + journal_article: field 'lay_summary' added (optional)         [safe]
  - journal_article: field 'conference_name' removed              [warn]
      1,247 works have non-empty values — data preserved in attrs
  ~ book: 'isbn' optional → required                              [warn]
      89 works have no isbn value — they will be grandfathered
  ! field name 'jounal_title' not in Go model                     [error]
```

Change classification:

| Change | Class | Notes |
|---|---|---|
| Field added | Safe | Existing records simply lack the field |
| Field removed | Warn | Data preserved in attrs; count of affected records shown |
| Required → optional | Warn | Validation loosened; no data impact |
| Optional → required | Warn | Count of records missing the field shown; existing records grandfathered |
| Kind deprecated | Warn | New works blocked; existing read-only |
| Field name not in Go type | Error | Blocked unconditionally — fix config or Go model first |

**`bbl profile check` — conflict resolution and enforcement gate:**

`bbl profile diff` tells you what changed. But for warnings, *the profile itself must
carry the resolution* — otherwise there is no way to proceed. The `accept:` block in
the profile file is the machine-readable acknowledgment of each warning. It is
committed to git, so the audit trail lives with the config, not in a runtime log.

```yaml
# profile.yaml

accept:
  - journal_article.conference_name.removed
      # 1,247 works have data in attrs; field no longer collected; data preserved
  - book.isbn.optional_to_required
      # 89 works without isbn are grandfathered; new submissions will require it

kinds:
  journal_article:
    fields:
      title:    { required: true }
      abstract: { optional: true }
      # conference_name removed — see accept block above
      volume:   { optional: true }
      issue:    { optional: true }
  book:
    fields:
      title: { required: true }
      isbn:  { required: true }
```

**Startup behavior:**

At startup bbl validates the loaded profile against the current DB state:

- Any `[error]` → refuse to start unconditionally (fix config or Go model).
- Any `[warn]` without a matching `accept:` entry → refuse to start.
- All clear → start normally.

No separate DB table or checksum needed. The profile file is self-contained: it
describes the structure and carries its own conflict resolutions in `accept:`. Startup
is the enforcement gate; `bbl profile diff` and `bbl profile check` are authoring
helpers (see below).

**`bbl profile` commands — authoring helpers:**

These commands help operators author a correct profile before deploying:

- **`bbl profile diff`** — read-only preview of data impact; use before editing.
- **`bbl profile check`** — validates the profile is conflict-free against the live
  DB (same check startup performs, without starting the server). Useful as a pre-deploy
  sanity check and in CI. Exit 0 = clean; non-zero = errors or unresolved warnings.

**Profile structure — one document, per-kind sections:**

A single YAML document with a section per work kind. Applied atomically as one
snapshot. Fields listed in declaration order, which is also the render order in forms.
Shared fields (title, abstract, contributors) are explicit in each kind's section —
no implicit inheritance.

```yaml
kinds:
  journal_article:
    fields:
      title:    { required: true }
      abstract: { optional: true }
      volume:   { optional: true }
      issue:    { optional: true }
  book:
    fields:
      title:    { required: true }
      isbn:     { optional: true }
```

**Kind deprecation:**

Removing a kind from the profile blocks creation of new works of that kind but does
not affect existing works. To handle existing works gracefully, kinds should be marked
deprecated rather than silently dropped:

```yaml
  conference_paper:
    deprecated: true   # no new works; existing works shown read-only
```

**Form generation:**

```
load_profile_for_kind(work.kind)
  → kind unknown (not in profile, not deprecated):
      render read-only attrs dump; show warning
  → kind deprecated:
      render read-only view of all stored attrs fields
  → kind active:
      for each field in profile[kind].fields (declaration order):
        render_field(name, required, work.attrs[name])
```

Fields absent from the profile are not rendered but their data in `attrs` is
untouched. Required fields are marked in the form. No separate ordering mechanism
is needed — the profile config is the authoritative render order.

### Ordering (fracdex)
Fracdex keys (`pos text NOT NULL COLLATE "C"`) support arbitrary insertion and reordering
with a single-row `UPDATE` — no renumbering of adjacent rows. This only matters when the
ordered set can be large or is reordered frequently.

Applied to: `bbl_work_contributors` (physics papers routinely have 3000+ authors;
mid-list insertion with `idx int` would touch thousands of rows) and `bbl_list_items`
(user-curated lists with no practical size bound).

Not applied to: `bbl_work_record_identifiers`, `bbl_work_organizations`,
`bbl_work_projects`, `bbl_work_files`, `bbl_work_rels` — all small, bounded sets where
renumbering on reorder is cheap and `idx int` is simpler to reason about.

Trade-off: no cheap positional access by rank. In practice the full ordered list is
always fetched and sliced in application code, so this is not a constraint for any of
the sets involved.

### Identifiers
The current schema stores identifiers as `(idx, scheme, val)` arrays per entity with no
source or validity. This makes temporal queries and reuse tracking impossible.

The greenfield approach normalizes identifiers into per-entity catalog tables
(`bbl_*_identifiers`) linked via join tables. One identifier value can appear on
multiple entities without data conflicts. Org identifiers carry `valid_from/to` to
handle reuse across splits and mergers; person, project, and work identifier join
tables are plain — wrong identifiers are deleted, not soft-revoked.

### Organization tree over time
`bbl_organization_rels` adds `valid_from/to` and `kind` so that mergers, splits, and
renames are first-class temporal facts rather than destructive mutations.
`kind` values like `merged_into` and `split_from` make the organizational history
navigable.

### Work contributors
A contributor known only by name — an external co-author not in the authority database —
is a valid, expected state. `person_identity_id` is intentionally nullable; there is
nothing to fix there.

Changing or nullifying `person_identity_id` is also a routine curator action (fixing a
wrong link). The forensic history of those changes is preserved in `bbl_mutations`, so
there is no data loss concern.

The actual gap is **rendering**: without a publication-time snapshot on the contributor
row, displaying a work requires joining to the current state of the linked identity,
which may differ from what was recorded at entry time (identity renamed, merged, or
deliberately unlinked).

The greenfield model adds `person_identity_snapshot jsonb` as a denormalized display
cache capturing name, role, and other attribution at time of entry:
- **Unlinked contributor**: snapshot holds the raw attributed name; `person_identity_id`
  is NULL and stays NULL.
- **Linked contributor**: snapshot holds attribution at link time. If the link is later
  changed or cleared, the snapshot provides stable display without touching `bbl_mutations`.

### Curation-only identities

The MDM records→identity model for people is designed for entities that arrive from
multiple external authority sources (ORCID, ISNI, Scopus). But some valid person
identities have no external source — a medieval historian, a pseudonymous author, a
historic local figure. Forcing a `source = 'manual'` record through the full MDM
stack just to create one identity is unnecessary machinery.

The pattern: a `bbl_person_identity` can have zero linked records. The identity row
itself is the authoritative edit surface. Curator commands write directly to `attrs`;
`provenance` is absent or `{}`. `ResolveIdentityProfile` is never called — there are
no records to aggregate from.

This is already compatible with the current schema. No structural change is needed;
it is an operational mode. The distinction:

| | Authority-aggregated | Curation-only |
|---|---|---|
| Records | ≥1 external source records | none |
| `attrs` | synthesised by resolution step | written directly by curator |
| `provenance` | `{field: source}` map | `{}` or absent |
| Re-resolution trigger | source record update | never (no sources) |
| Match candidates | generated by matching agents | not applicable |

Curation-only identities live in the same tables and participate in the same
contributor links. The only difference is that the resolution machinery is never
invoked for them.

The people model degrades gracefully: an identity that starts as curation-only can
gain external records later (a researcher claims their ORCID), at which point it
transitions to authority-aggregated without any row migration.

Note: `bbl_organizations` is always curator-managed — no records layer, no resolution
step. It is structurally the simpler model for the same reason: orgs arrive from at
most one source at a time and do not require cross-source deduplication.

### Mutation model

**Mutations as first-class serializable values**

The prototype has two overlapping concepts — `Action` (outer envelope in `repo.go`)
and `WorkChanger` (inner work-specific mutation) — with duplicated JSON
deserialization at both levels. The greenfield model collapses these into a single
`Mutation` type: a named, serializable unit of intent.

```go
type Mutation struct {
    Name       string          // "SetTitle" | "PublishWork" | "AddContributor" | ...
    EntityType string          // "work" | "person_identity" | "organization" | ...
    EntityID   uuid.UUID
    Args       json.RawMessage
}
```

Mutations are plain data. They can be built inline, deserialized from JSON (API,
batch file), constructed by a harvester pipeline, or assembled by a batch builder —
and then passed to `AddRev` unchanged.

```go
AddRev(ctx, userID, source, []Mutation) (revID uuid.UUID, error)
```

**MutationImpl — declare needs, then apply**

Each named mutation is backed by a registered `MutationImpl`:

```go
type MutationImpl interface {
    // Needs declares what state must be pre-fetched. Computable from args alone.
    Needs(m Mutation) MutationNeeds

    // Apply is pure: no DB access. Receives pre-fetched state, returns diff.
    Apply(state MutationState, m Mutation) (Diff, error)
}

type MutationNeeds struct {
    WorkIDs    []uuid.UUID
    PersonRefs []Ref
    // ... other entity types
}

type MutationState struct {
    Works   map[uuid.UUID]*Work
    Persons map[uuid.UUID]*PersonIdentity
    // ...
}

RegisterMutation(name string, impl MutationImpl)
```

**AddRev — two-phase, two round-trips regardless of mutation count**

```
1. Call Needs() on all mutations → union into one MutationNeeds
2. One batch read per entity type (WHERE id = ANY($1)) — pgx pipeline
3. Build MutationState from results
4. Call Apply() on each mutation → collect Diffs
5. One batch write: entity updates + INSERT INTO bbl_mutations rows
```

A rev with 50 mutations costs the same in DB round-trips as a rev with 1. The work
is in the Go layer (unioning needs, applying diffs), not in the DB.

`Apply` is pure and has no DB dependency — fully testable without a connection.

**RegisterMutation** replaces `WorkChangers` map and `Action` switch. One registry,
one deserialization path, no nesting.

### bbl_mutations table
The current `CHECK` sum constraint works but is brittle for adding new entity types.
The greenfield model uses explicit `entity_type text` + `entity_id uuid` columns plus
an `op_type` field, and adds `name` (the registered `MutationImpl`). This makes
cross-entity queries natural and is easier to extend.

#### diff envelope

Every `diff` value follows a single command envelope:

```json
{
  "name": "SetTitle",
  "args": { "title": "New Title" },
  "prev": { "title": "Old Title" }
}
```

- **`name`** — the command name, matching the Repository method that produced the change.
- **`args`** — the new values for the fields the command writes. Only the fields actually
  touched are present; unaffected fields are absent.
- **`prev`** — the prior values of those same fields, captured before the write. Present
  for all `update` ops. Omitted (or `{}`) for `create`; `args` is omitted for `delete`.

In practice two kinds of commands write to this table:

**Discrete curator/user commands** — each has a specific name and a small, bounded
`args`/`prev` set:
```json
{ "name": "SetTitle",        "args": {"title": "New"},         "prev": {"title": "Old"} }
{ "name": "Publish",         "args": {"status": "public"},     "prev": {"status": "draft"} }
{ "name": "AddContributor",  "args": {"pos": "a", "name": "J. Doe"} }
```

**System bulk commands** — harvesters and importers that push a partial or complete new
document version use the same envelope with a conventional name:
```json
{ "name": "MergeAttrs",   "args": {"title": "X", "year": 2024}, "prev": {"title": "Y", "year": 2023} }
{ "name": "ReplaceAttrs", "args": {<full attrs>},               "prev": {<full attrs>} }
```

`MergeAttrs` is a partial update (only supplied fields change); `ReplaceAttrs` is a
full swap of the `attrs` blob. Both exclude protected fields (`status`, `delete_kind`,
`deleted_at`, `deleted_by_id`, `attrs_purged_at`) — those fields are only ever written
by their own named commands.

The human-vs-system distinction is already carried by `bbl_revs`: `user_id` is non-NULL
for human actions, `source` is non-NULL for system actions. No `diff_kind` discriminator
is needed in the diff body itself.

`prev` enables GDPR field-level reasoning: to purge a specific personal data field (e.g.
a name value) from the change history, scan `diff->>'name'` and zero out only the
relevant key in `args` and `prev` — no need to inspect or rewrite the full `attrs` blob.

### Representations
`bbl_work_representations` is a precomputed serialization cache keyed by `(work_id, scheme)`.
`work_version` captures `bbl_works.version` at render time; a catbird background task
detects stale rows with `WHERE work_version < bbl_works.version` and queues re-renders.

`record_sha256` detects no-op re-renders: `UpsertWorkRepresentation` hashes the
incoming bytes, compares to the stored hash, and skips the write when they match.
`updated_at` only advances on content change — a curator editing a field that a scheme
does not expose produces no spurious bump and triggers no downstream OAI-PMH or webhook
consumers.

`bbl_work_collection_works` links works to named collections (OAI-PMH sets, open access
subsets, faculty feeds). Collections are administratively defined, distinct from
user-curated `bbl_lists`. The representation is looked up at serve time by scheme.

### bbl_revs source field
Automated imports should be distinguishable from human edits in the audit log.
`source` on `bbl_revs` links to `bbl_sources`, making this a first-class fact.

### User sources, staleness, and auth methods

**User sources and staleness detection**

Users can arrive from multiple sources: recurring directory sweeps (LDAP, SCIM),
one-time bulk imports, or manual admin creation. `bbl_user_sources` tracks provenance
per `(user, source)` and drives staleness detection without requiring the ingest layer
to hold a full set in memory.

The ingest layer stamps `last_seen_at` for each user yielded during a sweep.
A Catbird job after the sweep queries:

```sql
SELECT user_id FROM bbl_user_sources
WHERE source = $1
  AND expires_at IS NOT NULL
  AND last_seen_at < $sweep_started_at
```

Users absent from the sweep get `deactivate_at` set. `expires_at IS NULL` marks
permanent rows (one-time imports, manually added users) — the staleness sweep skips
these.

`UserSource` itself is a pure stream; it has no knowledge of staleness. The ingest
layer owns the `UpsertUserSource` stamp; Catbird owns the deactivation sweep. The
source just harvests.

**Pluggable auth providers**

`bbl_user_auth_methods` associates users with named auth provider instances —
`"ugent_oidc"`, `"orcid_oidc"`, `"magic_link"` — not generic protocol types. Using
named instances allows a user to have multiple OIDC providers simultaneously.

```go
type AuthProvider interface {
    ID() string
    BeginAuth(w http.ResponseWriter, r *http.Request) error
    CompleteAuth(w http.ResponseWriter, r *http.Request) (Claims, error)
}

RegisterAuthProvider(provider AuthProvider)
```

Login flow: look up `bbl_user_auth_methods` by provider + identifier, dispatch to the
registered provider. When a `UserSource` harvests a user, the ingest layer
auto-associates the auth provider configured for that source — no manual wiring
required for directory-sourced users.

### User ↔ person identity link
The current implicit model derives the user↔person connection by matching shared
identifiers at query time. This has several problems:
- Identifier drift on either side silently creates or breaks associations.
- `bbl_user_identifiers` conflates two distinct concerns: SSO login tokens (used to
  match an incoming OIDC claim to a `bbl_users` row) and authority matching signals
  (used to find a `bbl_person_identity`). These have different owners, different
  update cadences, and different trust levels.
- No audit trail for when or how the association was established.

The greenfield model makes the link explicit:
- `bbl_users.person_identity_id uuid UNIQUE` — nullable FK to `bbl_person_identities`.
  Nullable covers service and admin accounts that have no research identity.
  `UNIQUE` ensures two accounts cannot both claim the same canonical person.
  `ON DELETE SET NULL` so retiring a person identity doesn't cascade-delete the user.
- The column is set and cleared through user management commands (`LinkUserToIdentity`,
  `UnlinkUserFromIdentity`), leaving a full audit trail in `bbl_user_events`.
- `bbl_user_identifiers` is scoped to authentication claims only (OIDC `sub`,
  `ugent_id` from LDAP, etc.). It plays no role in person authority matching.
  Authority matching signals live exclusively in `bbl_person_record_identifiers`.

### Permissions

The core problem: answering "what can user X do?" currently requires inspecting
`bbl_users.role`, `bbl_work_permissions`, and implicit creator logic separately.
There is no single place to look.

**`bbl_grants` — one table, one query**  
All grants — global roles, org-scoped curation rights, project-scoped rights, and
entity-level ad-hoc access — live in a single `bbl_grants` table keyed by
`(user_id, kind, scope_type, scope_id)`. A complete picture of any user's active
rights is always one query:
```sql
SELECT * FROM bbl_grants
WHERE user_id = $1
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY scope_type NULLS FIRST, scope_id;
```
This is also the backoffice "access overview" view: one page per user with all their
grants, filterable by scope and kind, with the granting curator and optional reason
visible on every row.

**`bbl_users.role` as the hard ceiling only**  
The global role on `bbl_users` is kept as a coarse administrative ceiling
(`admin` | `user`) that guards system-level actions (running migrations, managing
sources, deactivating users). It does not replace `bbl_grants` — it just provides a
fast guard that short-circuits before the grants table is consulted. Curation rights
are expressed exclusively through `bbl_grants`.

**Scope types and inheritance**  
- `NULL` (global) — applies everywhere, used sparingly for institution-wide curators.
- `'organization'/{id}` — covers all works and people linked to that org and its
  descendants. Inheritance is a query-time tree walk on `bbl_organization_rels`;
  it is not denormalized.
- `'project'/{id}` — covers all works in that project.
- `'work'/{id}`, `'person_identity'/{id}`, etc. — entity-level, covers ad-hoc and
  ownership grants. Replaces `bbl_work_permissions`.

**Explicit ownership**  
`kind='owner'` with `scope_type='work'` replaces implicit creator rights. Ownership is
transferable (`UPDATE bbl_grants SET user_id = ...`), revocable, and fully audited via
`bbl_mutations`. `created_by_id` on the work row remains a pure creation audit column.

**Proxy delegation**
`bbl_user_proxies` handles full-person delegation (leave coverage). A proxy inherits
the proxied user's entire effective grant set dynamically for the duration of the
window — no grant copying required.

**`note` field**  
Every grant row carries an optional `note text`. This turns the grants table into a
self-documenting audit log: "temporary edit access while PI is on leave",
"owner transferred at PI request". Important for compliance and for curators reviewing
access.

**Referential integrity without FKs**  
`scope_id` is a polymorphic reference — it points to different tables depending on
`scope_type`. PostgreSQL cannot express this as a FK, so the column carries no
constraint. This is safe because a grant pointing to a deleted entity cannot grant
access to anything (the entity no longer exists). Orphaned rows are noise, not a
vulnerability.

Consistency is maintained in the entity delete command (`AddRev`): deleting a work,
person identity, organization, or project includes a `DELETE FROM bbl_grants WHERE
scope_type = $type AND scope_id = $id` as part of the same transaction. Since all
mutation is concentrated in a small number of command functions (the existing pattern),
every delete path passes through the same code and the cleanup cannot be silently
skipped. The grant deletions are recorded in `bbl_mutations` under the same rev as the
entity delete, keeping the audit trail intact.

A periodic catbird sweep acts as a safety net in case of direct-SQL operations or
migration bugs, but is not the primary consistency mechanism.

### Work source provenance
Works can be contributed by multiple independent sources (WoS, ORCID, manual entry,
future harvesters). After a candidate is accepted and a work is created, the only
existing link is `bbl_work_candidates.work_id` — a candidate-side reference that
becomes stale if attrs are cleared, and silently disappears if a second source later
brings in the same work.

`bbl_work_sources` is a thin provenance table: one row per `(work, source)`, recording
the `source_record_id` and a back-reference to the candidate that triggered the last
ingestion. It supports re-sync (look up the source record ID to pull an updated
version), multi-source merge (a second candidate for the same DOI adds a second row
rather than overwriting the first), and attribution display ("imported from Web of
Science · last updated 2025-01-10").

This is intentionally lighter than the full MDM records/identities model used for
people. Work deduplication happens at the candidate stage via identifier matching;
no separate records layer is needed.

### MergeAttrs and source precedence

Once a work is accepted, `bbl_works.attrs` is the curator's authoritative version —
not an aggregate of source inputs. A subsequent harvester `MergeAttrs` must not
silently overwrite curator edits. Two complementary mechanisms enforce this:

**C — No direct `MergeAttrs` on accepted works**
Harvesters update the source candidate record rather than `bbl_works.attrs` directly.
A separate resolution step (human or automated) synthesises `bbl_works.attrs` from
the candidate set. This is the structural safeguard.

**B — Curator-locked fields (`attrs_locked_fields text[]`)**
A curator can explicitly lock individual fields (e.g. `["title", "year"]`). Any
`MergeAttrs` that does reach an accepted work (e.g. a privileged admin path) silently
skips locked fields. No priority ordering required.

C is the default; B is the per-field escape hatch when automated resolution still
needs to refresh most fields but the curator has fixed specific ones.

---

## Work candidates

Candidates are possible works collected by automated agents (e.g. Web of Science, arXiv
harvesting). They are not works until explicitly accepted. The current approach of
modeling them as `bbl_works` with a special status has several problems:

- Pollutes the works table and all its indexes with large volumes of low-quality rows.
- All work queries must filter out candidates; easy to miss.
- Metadata goes stale; you can't prune the record without losing the rejection history.
- Candidates get indexed in OpenSearch unnecessarily.

### Model: lean staging with identity hooks

Candidates are unresolved staging data. Resolution of contributors, orgs, and projects
happens at acceptance time — not at ingest. The only structured sub-tables are those
needed for duplicate detection and backoffice review queue routing.

```sql
-- Candidates: thin staging table. attrs is purgeable raw payload.
-- Rejected rows stay (status='rejected', attrs cleared) as implicit tombstones —
-- no separate rejections table needed unless hard-delete volume becomes a concern.
CREATE TABLE bbl_work_candidates (
    id               uuid PRIMARY KEY,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    status           text NOT NULL DEFAULT 'pending',  -- pending | accepted | rejected
    confidence       numeric,
    attrs            jsonb NOT NULL DEFAULT '{}',
    fetched_at       timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at       timestamptz,
    decided_at       timestamptz,
    decided_by_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id   uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    work_id          uuid REFERENCES bbl_works (id) ON DELETE SET NULL,
    UNIQUE (source, source_record_id)
);

CREATE INDEX ON bbl_work_candidates (status);
CREATE INDEX ON bbl_work_candidates (source, status);
CREATE INDEX ON bbl_work_candidates (expires_at) WHERE status = 'pending';
CREATE INDEX ON bbl_work_candidates (work_id) WHERE work_id IS NOT NULL;

-- Extracted identifiers: enables duplicate detection against bbl_work_identifiers.
-- Refreshed independently from attrs; survives attrs pruning.
CREATE TABLE bbl_work_candidate_identifiers (
    candidate_id uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    scheme       text NOT NULL,
    value   text NOT NULL,
    PRIMARY KEY (candidate_id, scheme)
);

CREATE INDEX ON bbl_work_candidate_identifiers (scheme, value);

-- Candidate → person identity suggestions.
-- Populated by harvesting/matching agents (catbird).
-- Powers the backoffice review queue: "show pending candidates for person X".
-- match_signal explains why the agent made the suggestion.
CREATE TABLE bbl_work_candidate_persons (
    candidate_id       uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    person_identity_id uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    confidence         numeric NOT NULL,
    match_signal       text,   -- e.g. 'orcid_exact', 'ugent_id_exact', 'name_fuzzy'
    PRIMARY KEY (candidate_id, person_identity_id)
);

CREATE INDEX ON bbl_work_candidate_persons (person_identity_id);
CREATE INDEX ON bbl_work_candidate_persons (person_identity_id, confidence);

-- Candidate → organization identity suggestions.
-- Powers the backoffice review queue: "show pending candidates for faculty X".
CREATE TABLE bbl_work_candidate_organizations (
    candidate_id    uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    confidence      numeric NOT NULL,
    match_signal    text,
    PRIMARY KEY (candidate_id, organization_id)
);

CREATE INDEX ON bbl_work_candidate_organizations (organization_id);
```

### Lifecycle

```
agent harvests record
    → check (source, source_record_id): skip if already accepted or rejected
    → INSERT INTO bbl_work_candidates (status='pending')
    → extract and upsert bbl_work_candidate_identifiers
    → score against person/org identities → insert bbl_work_candidate_persons/organizations

curator reviews backoffice queue (filtered by person or org):
    accept → create bbl_work via AddRev, set status='accepted', set work_id
    reject → set status='rejected', clear attrs to '{}'

background job (catbird):
    → refresh attrs + re-extract identifiers where expires_at < now()
    → re-score person/org suggestions on new imports or after identity changes
```

### Key design decisions

- **Lean staging contract**: candidates are unresolved. Contributor/org/project resolution
  happens at acceptance, not ingest. No sub-tables for those — keeps the model honest
  about what staging data actually is.
- **Identity hooks without full resolution**: `bbl_work_candidate_persons` and
  `bbl_work_candidate_organizations` give just enough structure to route the review queue
  without rebuilding the full work relational model.
- **Implicit tombstone**: rejected rows stay with `attrs = '{}'`. A separate rejections
  table is only needed if hard-deleting rejected rows becomes necessary at scale.
- **`expires_at`**: agent-controlled staleness signal; catbird worker uses it to schedule
  refreshes.
- **No FK from work to candidate**: work is authoritative after acceptance; link is
  candidate-side only.

## Open questions

- **`bbl_people` migration**: treat existing rows as curation-only person identities
  (no records), or run old and new models in parallel during transition?
- **Org name snapshot at link time**: `bbl_work_organizations` links a work to an org by
  UUID. If the org is later renamed (same UUID, updated `attrs.names`), the link still
  resolves but the name shown will differ from the name at time of authorship. Options:
  (a) accept it — org renames are rare and current name is good enough for display;
  (b) add `organization_snapshot jsonb` to `bbl_work_organizations`, same pattern as
  `person_identity_snapshot` on contributors. Decide before implementation.
- **`bbl_mutations` entity_id type**: `uuid` works for all current entities but breaks if
  non-UUID entity types are added later.
- **`bbl_mutations` table size and partitioning**: at institutional scale this table will
  be very large. Three options worth evaluating before the first migration: (1) keep the
  `bigserial` PK and add time-range partitioning — global monotonic order is preserved
  via a shared sequence, old partitions detach to cold storage, but per-entity queries
  scan all partitions without a time filter; (2) drop the sequence and use
  `(rev_id, entity_type, entity_id)` as PK, ordering via `bbl_revs.created_at` —
  freely partitionable, no sequence bottleneck, but "all changes since cursor X" becomes
  a join; (3) keep `bbl_mutations` as-is and add a separate lightweight `bbl_events`
  outbox table with a `bigserial` for external consumers (event stream, webhooks) that
  can be pruned after acknowledgement — `bbl_mutations` stays large but is only queried
  per-entity. Option 2 is preferred unless an external event stream cursor is needed,
  in which case option 3 adds that without polluting the audit table.
- **Work identifier exclusivity**: enforce unique active DOI ownership at DB level once
  policy is defined?
- **File access control granularity**: per-file (as sketched) or per-work-file-group?
- **Events/conferences as a structured entity**: currently a conference venue can be stored as a free-text field on a work (no schema change needed for basic use). If conferences need deduplication across sources, the same records/identities split used for orgs/persons applies — one flat table vs six. Journal/series editorship is a clean add via `bbl_person_affiliations.kind` or a dedicated link table. Per-work editorship already works via `bbl_work_contributors.role`. The authority model question (flat vs MDM) should be decided before any conference table is added, because it determines the shape of the work-event link and whether a venue-name snapshot is needed on that link row.
- **attrs fields worth promoting to SQL columns**: search is handled by the search engine so SQL queryability of metadata is not the primary driver. However, a small number of `attrs` fields may still be worth promoting to real columns for constraint enforcement or partial-index efficiency — candidates are `publication_year` (range filters in backoffice), `language` (coverage reporting), and potentially `access_kind` at the work level (distinct from file-level access). Decide per field whether the benefit outweighs the migration cost.
