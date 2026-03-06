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
| Representations | Hard FK to `bbl_works` — no equivalent cache for persons, organizations, or projects even though their serialized forms (VCARD, org XML, CSL) can be equally expensive to compute |
| Representations | No staleness signal — no way to know if a cached representation is still current without comparing to the entity's `updated_at` |
| Work permissions | No PK, no timestamps, no expiry |
| Permissions | Rights are scattered across `bbl_users.role`, `bbl_work_permissions`, and implicit creator logic — no single query can show what a user is allowed to do |
| Permissions | Global role (`admin \| curator \| user`) is unscoped — a curator has equal rights over all works and people; in practice curation is usually limited to a faculty or department |
| Permissions | Implied creator ownership is not schema-represented — can't be transferred, queried, or revoked without touching application logic |
| Permissions | Proxy delegation is all-or-nothing — user A gets full impersonation of user B with no way to scope it to a subset of their entities |
| Permissions | Ad-hoc grants (`bbl_work_permissions`) apply only to works — no equivalent for person records, projects, or other entities |
| User proxies | No temporal bounds, no reason |
| `bbl_revs` | No source/context field (which system made this change?) |
| `bbl_changes` | `diff jsonb` is opaque; no `op_type` (create/update/delete) |
| `bbl_changes` | SUM check constraint is fragile; no explicit `entity_type` discriminator |
| Projects | No temporal bounds at schema level (start/end dates only in attrs) |
| Lists | `bbl_list_items` hard-codes `work_id` — no support for lists of persons, organizations, or projects |
| Lists | A list has no declared type constraint — nothing prevents mixing entity types unless enforced in application code |
| General | No source registry / import precedence table |
| General | `idx int` ordering requires renumbering all following rows on any insertion or reorder — every user-reorderable list pays that cost |
| Work rels | Directional — querying all works related to X requires checking both `work_id = X` and `rel_work_id = X`; `kind` semantics (symmetric vs asymmetric) are undocumented at schema level |
| Works | No tombstone metadata — `status='deleted'` provides the tombstone but there is no `deleted_at` / `deleted_by_id` to record when or by whom a work was withdrawn or retracted; no `delete_kind` to distinguish routine withdrawal from a legally-mandated takedown (GDPR, patent, right to be forgotten), and no record of when personal data was purged from `attrs` |
| `bbl_changes` | The audit trail is an invariant everywhere, but GDPR right-to-erasure and right-to-be-forgotten may legally oblige purging change history rows for a specific entity too — `diff` can contain personal data captured at the time of each mutation |
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
-- Higher priority = wins field-level conflicts during resolution.
-- ============================================================

CREATE TABLE bbl_sources (
    id          text PRIMARY KEY,              -- e.g. 'ugent_ldap', 'orcid', 'plato', 'manual'
    label       text NOT NULL,
    priority    int NOT NULL DEFAULT 0,        -- higher = more authoritative
    description text
);

-- ============================================================
-- USERS
-- Application accounts.
-- person_identity_id is the explicit link to the canonical person authority record.
-- It is nullable (service/admin accounts have no person identity) and unique
-- (two accounts cannot claim the same identity).
-- Set and removed through AddRev commands; audited in bbl_changes.
-- Forward FK added via ALTER TABLE after bbl_person_identities is defined below.
-- ============================================================

CREATE TABLE bbl_users (
    id                 uuid PRIMARY KEY,
    version            int NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id      uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id      uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    username           text NOT NULL UNIQUE,
    email              text NOT NULL COLLATE bbl_case_insensitive,
    name               text NOT NULL,
    role               text NOT NULL,               -- 'admin' | 'curator' | 'user'
    deactivate_at      timestamptz,
    person_identity_id uuid UNIQUE  -- FK added below: REFERENCES bbl_person_identities (id) ON DELETE SET NULL
);

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
CREATE INDEX ON bbl_grants (user_id) WHERE expires_at IS NULL;  -- active global/permanent grants
CREATE INDEX ON bbl_grants (expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================
-- ORGANIZATIONS
-- Two-layer model: source records + canonical identities.
-- Mirrors the people dedup model.
-- ============================================================

-- Canonical institutional identity (the durable "real-world org").
-- Survives renames, reorganizations, and mergers.
CREATE TABLE bbl_organization_identities (
    id                  uuid PRIMARY KEY,
    version             int NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind                text NOT NULL,          -- 'faculty' | 'department' | 'research_group' | ...
    resolved_attrs      jsonb NOT NULL DEFAULT '{}',
    resolved_provenance jsonb NOT NULL DEFAULT '{}'
);

-- Source avatars: one row per imported or manually created org payload.
CREATE TABLE bbl_organization_records (
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

-- Record → identity membership with audit.
CREATE TABLE bbl_organization_identity_members (
    identity_id        uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE CASCADE,
    record_id          uuid NOT NULL REFERENCES bbl_organization_records (id) ON DELETE CASCADE,
    PRIMARY KEY (identity_id, record_id),
    status             text NOT NULL DEFAULT 'active',   -- active | pending | rejected
    link_kind          text NOT NULL,                    -- auto | manual | imported
    confidence         numeric,
    decided_by_user_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id     uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE INDEX ON bbl_organization_identity_members (record_id);
CREATE INDEX ON bbl_organization_identity_members (status);

-- Identifiers with normalized value and temporal validity.
-- One identifier may move between orgs over time (identifier reuse).
CREATE TABLE bbl_organization_identifiers (
    id         uuid PRIMARY KEY,
    scheme     text NOT NULL,
    value_norm text NOT NULL,
    issuer     text NOT NULL DEFAULT '',
    UNIQUE (scheme, value_norm, issuer)
);

CREATE INDEX ON bbl_organization_identifiers (scheme, value_norm);

CREATE TABLE bbl_organization_record_identifiers (
    record_id     uuid NOT NULL REFERENCES bbl_organization_records (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_organization_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (record_id, identifier_id),
    valid_from    timestamptz,
    valid_to      timestamptz,
    revoked_at    timestamptz
);

CREATE INDEX ON bbl_organization_record_identifiers (identifier_id);

-- Temporal hierarchical relationships between organization identities.
-- Supports parent/child, mergers, splits, and renames over time.
-- kind: 'part_of' | 'merged_into' | 'split_from' | 'successor_of'
CREATE TABLE bbl_organization_rels (
    id               uuid PRIMARY KEY,
    organization_id  uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE CASCADE,
    rel_organization_id uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE CASCADE,
    kind             text NOT NULL,
    valid_from       timestamptz,
    valid_to         timestamptz,
    decided_rev_id   uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    CHECK (organization_id <> rel_organization_id)
);

CREATE INDEX ON bbl_organization_rels (organization_id);
CREATE INDEX ON bbl_organization_rels (rel_organization_id);
-- Current active relationships:
CREATE INDEX ON bbl_organization_rels (organization_id) WHERE valid_to IS NULL;

-- ============================================================
-- PEOPLE
-- Approach: MDM consolidation with durable source records.
--
-- person_records  = immutable-ish source avatars (one per import payload)
-- person_identities = canonical golden records (one per real-world person)
-- person_identity_members = the consolidation link; carries process metadata
--
-- [source A]──┐
-- [source B]──┼──► person_records ──► person_identity_members ──► person_identities
-- [manual  ]──┘                            (link process)            (golden record)
--
-- A record belongs to at most one active identity.
-- Enforced in command logic, not a hard DB constraint (policy not yet stable).
-- All mutations go through AddRev; decided_rev_id provides the audit trail.
-- ============================================================

CREATE TABLE bbl_person_identities (
    id                  uuid PRIMARY KEY,
    version             int NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    resolved_attrs      jsonb NOT NULL DEFAULT '{}',
    resolved_provenance jsonb NOT NULL DEFAULT '{}'
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

CREATE TABLE bbl_person_identity_members (
    identity_id        uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    record_id          uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    PRIMARY KEY (identity_id, record_id),
    status             text NOT NULL DEFAULT 'active',
    link_kind          text NOT NULL,
    confidence         numeric,
    decided_by_user_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id     uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE INDEX ON bbl_person_identity_members (record_id);
CREATE INDEX ON bbl_person_identity_members (status);

CREATE TABLE bbl_person_identifiers (
    id         uuid PRIMARY KEY,
    scheme     text NOT NULL,
    value_norm text NOT NULL,
    issuer     text NOT NULL DEFAULT '',
    UNIQUE (scheme, value_norm, issuer)
);

CREATE INDEX ON bbl_person_identifiers (scheme, value_norm);

CREATE TABLE bbl_person_record_identifiers (
    record_id     uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_person_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (record_id, identifier_id),
    confidence    numeric,
    valid_from    timestamptz,
    valid_to      timestamptz,
    revoked_at    timestamptz
);

CREATE INDEX ON bbl_person_record_identifiers (identifier_id);

-- Temporal affiliations between person identities and organization identities.
-- Replaces the flat bbl_person_organizations table.
CREATE TABLE bbl_person_affiliations (
    id              uuid PRIMARY KEY,
    person_id       uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE CASCADE,
    role            text,                   -- e.g. 'researcher', 'professor', 'phd_student'
    valid_from      timestamptz,
    valid_to        timestamptz,
    source          text REFERENCES bbl_sources (id),
    created_rev_id  uuid REFERENCES bbl_revs (id) ON DELETE SET NULL
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
    decided_rev_id     uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
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
    starts_on     date,
    ends_on       date,
    attrs         jsonb NOT NULL DEFAULT '{}'
);

-- Normalized project identifiers with source and validity.
CREATE TABLE bbl_project_identifiers (
    id         uuid PRIMARY KEY,
    scheme     text NOT NULL,
    value_norm text NOT NULL,
    issuer     text NOT NULL DEFAULT '',
    UNIQUE (scheme, value_norm, issuer)
);

CREATE TABLE bbl_project_record_identifiers (
    project_id    uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_project_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, identifier_id),
    source        text REFERENCES bbl_sources (id),
    valid_from    timestamptz,
    valid_to      timestamptz
);

CREATE INDEX ON bbl_project_record_identifiers (identifier_id);

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
    created_rev_id uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    UNIQUE (person_id, project_id, role)
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
-- their bbl_changes rows can be deleted in the same transaction with no special ceremony.
--
-- For legally-mandated takedowns (GDPR, patent, right to be forgotten) the row
-- stays but attrs is purged (set to '{}'); attrs_purged_at records when this happened.
-- For GDPR erasure / right-to-be-forgotten specifically, bbl_changes rows for the
-- entity may also need to be deleted (diff can contain personal data). This is tracked
-- separately — changes_purged_at in bbl_work_takedowns records that obligation.
-- delete_kind distinguishes routine editorial events from legal obligations:
--   'withdrawn'  = author/editor request post-publication; tombstone only, attrs kept, changes kept
--   'retracted'  = post-publication integrity issue; tombstone only, attrs kept, changes kept
--   'takedown'   = legal obligation; attrs purged, bbl_work_takedowns row required
--                  changes history may also be purged depending on legal_basis
CREATE TABLE bbl_works (
    id             uuid PRIMARY KEY,
    version        int NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind           text NOT NULL,     -- 'journal_article' | 'book' | 'dataset' | ...
    subkind        text,
    status         text NOT NULL,     -- 'draft' | 'suggestion' | 'public' | 'deleted'
    delete_kind    text,              -- 'withdrawn' | 'retracted' | 'takedown'; set with status='deleted'
    deleted_at     timestamptz,       -- set when status transitions to 'deleted'
    deleted_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs_purged_at timestamptz,      -- set when attrs was legally scrubbed; implies attrs = '{}'
    attrs          jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX ON bbl_works (status);
CREATE INDEX ON bbl_works (status) WHERE status = 'deleted' AND attrs_purged_at IS NULL;  -- tombstones with content

-- Legal takedown record. One row per takedown decision.
-- Tracks the legal basis and request provenance; this row survives even after attrs
-- is purged and is itself subject to data retention policies.
-- legal_basis: 'gdpr_erasure' | 'right_to_be_forgotten' | 'patent' | 'court_order' | 'other'
CREATE TABLE bbl_work_takedowns (
    id             uuid PRIMARY KEY,
    work_id        uuid NOT NULL REFERENCES bbl_works (id),  -- intentionally no CASCADE
    legal_basis    text NOT NULL,
    reference      text,              -- external case/ticket/dossier reference
    requested_at   timestamptz NOT NULL,
    requested_by   text,              -- name or org of requesting party (free text, may be redacted)
    decided_at     timestamptz,
    decided_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    attrs_purged_at   timestamptz,      -- when bbl_works.attrs was set to '{}'
    changes_purged_at timestamptz,      -- when bbl_changes rows for this work were deleted;
                                        -- only required for gdpr_erasure / right_to_be_forgotten;
                                        -- NULL = changes history retained (patent, court_order, etc.)
    notes          text               -- internal curator notes; not exposed publicly
);

CREATE INDEX ON bbl_work_takedowns (work_id);
CREATE INDEX ON bbl_work_takedowns (legal_basis);

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

-- Work identifiers: normalized, with source and validity.
CREATE TABLE bbl_work_identifiers (
    id         uuid PRIMARY KEY,
    scheme     text NOT NULL,
    value_norm text NOT NULL,
    issuer     text NOT NULL DEFAULT '',
    UNIQUE (scheme, value_norm, issuer)
);

CREATE TABLE bbl_work_record_identifiers (
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_work_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (work_id, identifier_id),
    idx           int NOT NULL,
    source        text REFERENCES bbl_sources (id),
    UNIQUE (work_id, idx)
);

CREATE INDEX ON bbl_work_record_identifiers (identifier_id);

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
    attrs                    jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (work_id, pos)
);

CREATE INDEX ON bbl_work_contributors (person_identity_id) WHERE person_identity_id IS NOT NULL;

-- Work ↔ organization links (affiliation at time of work, not temporal).
CREATE TABLE bbl_work_organizations (
    work_id         uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    idx             int NOT NULL,
    organization_id uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE RESTRICT,
    role            text,
    PRIMARY KEY (work_id, idx),
    UNIQUE (work_id, organization_id)
);

CREATE INDEX ON bbl_work_organizations (organization_id);

-- Work ↔ project links.
CREATE TABLE bbl_work_projects (
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    idx        int NOT NULL,
    project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE RESTRICT,
    PRIMARY KEY (work_id, idx),
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
    idx          int NOT NULL,
    kind         text NOT NULL,
    rel_work_id  uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    PRIMARY KEY (work_id, idx),
    CHECK (work_id <> rel_work_id)
);

CREATE INDEX ON bbl_work_rels (rel_work_id);

-- Files: includes checksum and upload status.
CREATE TABLE bbl_work_files (
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    idx           int NOT NULL,
    object_id     uuid NOT NULL,
    name          text NOT NULL,
    content_type  text NOT NULL,
    size          int NOT NULL,
    sha256        text,          -- populated after upload confirmation
    upload_status text NOT NULL DEFAULT 'pending',  -- pending | complete | failed
    PRIMARY KEY (work_id, idx)
);

-- Per-work-file access control (open | restricted | embargo | closed).
CREATE TABLE bbl_work_file_access (
    work_id       uuid NOT NULL,
    idx           int NOT NULL,
    kind          text NOT NULL DEFAULT 'open',
    embargo_until timestamptz,
    FOREIGN KEY (work_id, idx) REFERENCES bbl_work_files (work_id, idx) ON DELETE CASCADE,
    PRIMARY KEY (work_id, idx)
);

-- Work-level permissions (bbl_work_permissions in the current schema) are subsumed
-- by bbl_grants with scope_type='work', scope_id=<work_id>.
-- Migration: INSERT INTO bbl_grants (user_id, kind, scope_type, scope_id, granted_at, expires_at)
--            SELECT user_id, kind, 'work', work_id, granted_at, expires_at FROM bbl_work_permissions;

-- ============================================================
-- REPRESENTATIONS & SETS
-- Generalized to cover any entity type, not just works.
-- entity_type mirrors bbl_changes: 'work' | 'person_identity' | 'organization_identity' | ...
-- entity_version is the entity.version captured at render time;
-- a background worker (catbird) can detect stale reps with:
--   SELECT r.* FROM bbl_representations r
--   JOIN bbl_works e ON r.entity_id = e.id
--   WHERE r.entity_type = 'work' AND r.entity_version < e.version
-- ============================================================

CREATE TABLE bbl_sets (
    id          uuid PRIMARY KEY,
    name        text NOT NULL UNIQUE,
    description text
);

CREATE TABLE bbl_representations (
    id             uuid PRIMARY KEY,
    entity_type    text NOT NULL,    -- 'work' | 'person_identity' | 'organization_identity' | ...
    entity_id      uuid NOT NULL,
    scheme         text NOT NULL,    -- 'oai_dc' | 'mods' | 'csl' | 'vcard' | ...
    record         bytea NOT NULL,
    entity_version int NOT NULL,     -- entity.version at render time; stale when entity.version > this
    updated_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (entity_type, entity_id, scheme)
);

CREATE INDEX ON bbl_representations (entity_type, entity_id);
CREATE INDEX ON bbl_representations (updated_at);

-- Sets group work representations for OAI-PMH set membership.
-- Keyed by representation id; set members are always entity_type='work' reps.
CREATE TABLE bbl_set_representations (
    set_id            uuid NOT NULL REFERENCES bbl_sets (id) ON DELETE CASCADE,
    representation_id uuid NOT NULL REFERENCES bbl_representations (id) ON DELETE CASCADE,
    UNIQUE (set_id, representation_id)
);

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

-- Changes: explicit entity_type discriminator + op_type instead of sum check.
-- Easier to query, easier to add new entity types, no CHECK arithmetic.
--
-- Legal exception: bbl_changes rows for a specific entity_id MAY be hard-deleted in
-- two sanctioned cases:
--   1. The work is still a draft (status='draft', never public): hard-delete the work
--      row and its changes in the same transaction; no special tracking needed.
--   2. A GDPR erasure or right-to-be-forgotten takedown is actioned on a public work:
--      diff can contain personal data captured at mutation time. The decision and
--      timestamp are recorded in bbl_work_takedowns.changes_purged_at before rows
--      are removed.
-- These are the only sanctioned hard-delete paths in the schema.
CREATE TABLE bbl_changes (
    id          bigserial PRIMARY KEY,
    rev_id      uuid NOT NULL REFERENCES bbl_revs (id),  -- no cascade: revs are immutable
    entity_type text NOT NULL,   -- 'user' | 'organization_identity' | 'organization_record'
                                 -- | 'person_identity' | 'person_record' | 'project' | 'work'
    entity_id   uuid NOT NULL,
    op_type     text NOT NULL,   -- 'create' | 'update' | 'delete'
    diff        jsonb NOT NULL
);

CREATE INDEX ON bbl_changes (rev_id);
CREATE INDEX ON bbl_changes (entity_type, entity_id);
CREATE INDEX ON bbl_changes (entity_id);

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
    name          text NOT NULL,
    public        boolean NOT NULL DEFAULT false,
    entity_type   text,                -- NULL = heterogeneous; set to lock list to one type
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL
);

CREATE INDEX ON bbl_lists (created_by_id);

CREATE TABLE bbl_list_items (
    list_id     uuid NOT NULL REFERENCES bbl_lists (id) ON DELETE CASCADE,
    entity_type text NOT NULL,   -- 'work' | 'person_identity' | 'organization_identity' | ...
    entity_id   uuid NOT NULL,
    pos         text NOT NULL COLLATE "C",
    UNIQUE (list_id, entity_type, entity_id),
    UNIQUE (list_id, pos)
);

CREATE INDEX ON bbl_list_items (entity_type, entity_id);

-- topic is a structured event kind e.g. 'work.updated', 'person_identity.merged'.
-- entity_type/entity_id optionally scope the subscription to a specific entity;
-- NULL entity_type = global topic subscription.
-- On entity delete, subscriptions scoped to that entity are removed in the same AddRev transaction.
--
-- Webhook delivery:
--   webhook_url NULL  = internal notification only (centrifugo / catbird); other webhook columns ignored.
--   webhook_secret    = used to produce an HMAC-SHA256 signature sent as X-Webhook-Signature on every
--                       delivery. Stored encrypted at rest. NULL = unsigned deliveries.
--   webhook_headers   = extra request headers as {"Header-Name": "value"} — use for Authorization:
--                       Bearer <token>, or any integration-specific header. Values stored encrypted.
--
-- Reliability:
--   enabled           = false suspends delivery without deletion; set by user or auto-set on excess failures.
--   suspended_at      = set automatically when failure_count exceeds the application threshold.
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
    entity_type         text,
    entity_id           uuid,
    webhook_url         text,                              -- NULL = internal only
    webhook_secret      text,                              -- HMAC-SHA256 signing secret; encrypted at rest
    webhook_headers     jsonb NOT NULL DEFAULT '{}',       -- extra headers; values encrypted at rest
    enabled             boolean NOT NULL DEFAULT true,
    suspended_at        timestamptz,                       -- set on auto-suspend from repeated failures
    failure_count       int NOT NULL DEFAULT 0,
    last_attempted_at   timestamptz,
    last_succeeded_at   timestamptz,
    created_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CHECK ((entity_type IS NULL) = (entity_id IS NULL)),
    CHECK (webhook_url IS NOT NULL OR (webhook_secret IS NULL AND webhook_headers = '{}'))
);

CREATE INDEX ON bbl_subscriptions (user_id);
CREATE INDEX ON bbl_subscriptions (topic);
CREATE INDEX ON bbl_subscriptions (entity_type, entity_id) WHERE entity_type IS NOT NULL;
CREATE INDEX ON bbl_subscriptions (topic, enabled) WHERE enabled = true;  -- delivery dispatcher

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
| `ResolveIdentityProfile(identity_id)` | Recompute `resolved_attrs` + `resolved_provenance` from active members |

---

## Repository — method surface

A single `Repository` backed by one PostgreSQL connection pool. All commands go through
`AddRev(ctx, userID, source, func(rev) error) (revID, error)` — one transaction, one
`bbl_revs` row, one or more `bbl_changes` rows. Queries are plain reads, no rev needed.

A split into multiple repos is not warranted: `AddRev` is a shared primitive, queries
join across entity boundaries, and `bbl_grants`/`bbl_changes` are genuinely cross-entity.

### Users

| Method | Type | Description |
|---|---|---|
| `GetUser(id)` | query | Fetch by primary key |
| `GetUserByUsername(username)` | query | Login lookup |
| `GetUserByIdentifier(scheme, val)` | query | Match incoming OIDC/LDAP claim |
| `ListUsers(opts)` | query | Paginated list with filters (role, deactivated, search) |
| `CreateUser(attrs)` | command | New application account |
| `UpdateUser(id, attrs)` | command | Profile update |
| `DeactivateUser(id)` | command | Set `deactivate_at`; does not hard-delete |
| `LinkUserToIdentity(userID, identityID)` | command | Set `bbl_users.person_identity_id`; clears previous link |
| `UnlinkUserFromIdentity(userID)` | command | Null out `person_identity_id` |
| `SetUserProxy(userID, proxyUserID, validFrom, validTo)` | command | Grant full proxy delegation |
| `RemoveUserProxy(id)` | command | Remove a proxy row by surrogate PK |

### Organizations

| Method | Type | Description |
|---|---|---|
| `GetOrganizationIdentity(id)` | query | Fetch canonical identity with resolved attrs |
| `GetOrganizationRecord(id)` | query | Fetch a source record |
| `ListOrganizationIdentities(opts)` | query | Paginated list with filters (kind, search) |
| `GetOrganizationTree(id, at)` | query | Ancestor/descendant walk via `bbl_organization_rels` at a point in time |
| `IngestOrganizationRecord(source, sourceRecordID, attrs)` | command | Import or refresh a source record |
| `CreateOrganizationIdentity()` | command | New canonical identity |
| `LinkOrgRecordToIdentity(recordID, identityID, linkKind, confidence)` | command | Attach record to identity |
| `UnlinkOrgRecordFromIdentity(recordID, identityID)` | command | Detach record; identity remains |
| `ResolveOrganizationProfile(identityID)` | command | Recompute `resolved_attrs` + `resolved_provenance` from active members |
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
| `ResolvePersonProfile(identityID)` | command | Recompute `resolved_attrs` + `resolved_provenance` from active members |
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
| `GetWorkHistory(id)` | query | `bbl_changes` rows for a work ordered by rev |
| `CreateWork(kind, attrs)` | command | New draft work; inserts `bbl_grants` owner row for creating user |
| `UpdateWork(id, attrs)` | command | Update metadata |
| `PublishWork(id)` | command | Transition `draft → public` |
| `WithdrawWork(id)` | command | Transition `public → deleted`, `delete_kind='withdrawn'` |
| `RetractWork(id)` | command | Transition `public → deleted`, `delete_kind='retracted'` |
| `DeleteDraftWork(id)` | command | Hard-delete; only valid while `status='draft'`; deletes `bbl_changes` rows in same transaction |
| `AddWorkContributor(workID, pos, attrs)` | command | Add contributor row; resolves identity link if possible |
| `UpdateWorkContributor(workID, pos, attrs)` | command | Update contributor attrs or identity link |
| `RemoveWorkContributor(workID, pos)` | command | Remove contributor by position |
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
| `PurgeWorkChanges(takedownID)` | command | Delete `bbl_changes` rows for the work, record `changes_purged_at`; only for gdpr/rtbf legal bases |

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
| `GetSubscriptionsForTopic(topic, entityType, entityID)` | query | Active subscriptions matching a fired event; used by catbird dispatcher |
| `CreateSubscription(userID, topic, entityType, entityID, webhookURL, secret, headers)` | command | New subscription |
| `UpdateSubscription(id, attrs)` | command | Change URL, headers, or topic |
| `EnableSubscription(id)` | command | Clear `suspended_at`, reset `failure_count`, set `enabled=true` |
| `DisableSubscription(id)` | command | Set `enabled=false` |
| `DeleteSubscription(id)` | command | Hard-delete |
| `RecordSubscriptionDelivery(id, succeeded, httpStatus, err)` | command | Update `failure_count`, `suspended_at`, `last_attempted_at`, `last_succeeded_at`; called by catbird job handler |

### Representations

| Method | Type | Description |
|---|---|---|
| `GetRepresentation(entityType, entityID, scheme)` | query | Fetch cached serialized form |
| `ListStaleRepresentations(entityType)` | query | Rows where `entity_version < current entity.version`; drives catbird batch re-render |
| `UpsertRepresentation(entityType, entityID, scheme, record, entityVersion)` | command | Insert or replace cache; sets `updated_at` |
| `DeleteRepresentation(entityType, entityID, scheme)` | command | Invalidate cache entry; next render re-populates it |

---

## Key design decisions and rationale

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

The greenfield approach normalizes identifiers into a shared catalog
(`bbl_*_identifiers`) with `value_norm`, then links records to them via a join table
with `valid_from/to` and `revoked_at`. One identifier can appear on multiple records
over time without data conflicts.

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
wrong link). The forensic history of those changes is preserved in `bbl_changes`, so
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
  changed or cleared, the snapshot provides stable display without touching `bbl_changes`.

### bbl_changes redesign
The current `CHECK` sum constraint works but is brittle for adding new entity types.
The greenfield model uses explicit `entity_type text` + `entity_id uuid` columns plus
an `op_type` field. This makes cross-entity queries natural and is easier to extend.

### Representations
The current `bbl_representations` table is a work-only precomputed serialization cache
primarily for OAI-PMH. The same need exists for other entities: a person identity's
VCARD, an organization's XML export, a CSL/BibTeX rendering of a work. Computing these
at request time is expensive enough to justify a cache.

The generalized model uses `(entity_type, entity_id, scheme)` — the same discriminator
pattern as `bbl_changes` — so all entity types share one table without a FK proliferation.
No hard FK to entity tables: the cache is intentionally soft-linked, and a missing or
stale row is never a correctness problem, only a performance one.

`entity_version` captures the entity's `version` counter at render time. Staleness
detection is a simple join:
```sql
SELECT r.entity_id FROM bbl_representations r
JOIN bbl_works e ON r.entity_id = e.id
WHERE r.entity_type = 'work' AND r.entity_version < e.version;
```
A catbird background task uses this to drive batch re-renders, keeping the cache
eventually consistent without synchronous coupling to the write path.

`bbl_set_representations` remains a work-only concept (OAI sets group works), but now
references the generalized table by `representation_id` without needing to change shape.

### bbl_revs source field
Automated imports should be distinguishable from human edits in the audit log.
`source` on `bbl_revs` links to `bbl_sources`, making this a first-class fact.

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
- The column is set and cleared through `AddRev` commands (`LinkUserToIdentity`,
  `UnlinkUserFromIdentity`), leaving a full audit trail in `bbl_changes`.
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
`bbl_changes`. `created_by_id` on the work row remains a pure creation audit column.

**Proxy scope**  
`bbl_user_proxies` handles full-person delegation (leave coverage). If scoped delegation
is needed later (a PA who should only curate works, not act on approvals), an
`optional scope_type/scope_id` pair can be added to `bbl_user_proxies` without
changing its existing rows.

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
skipped. The grant deletions are recorded in `bbl_changes` under the same rev as the
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
    value_norm   text NOT NULL,
    PRIMARY KEY (candidate_id, scheme)
);

CREATE INDEX ON bbl_work_candidate_identifiers (scheme, value_norm);

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
    organization_id uuid NOT NULL REFERENCES bbl_organization_identities (id) ON DELETE CASCADE,
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

- **`bbl_people` / `bbl_organizations` migration**: treat existing rows as `source = 'manual'`
  records, or run old and new models in parallel during transition?
- **Organization dedup**: apply the same match candidate + scores tables as for people?
- **`bbl_changes` entity_id type**: `uuid` works for all current entities but breaks if
  non-UUID entity types are added later.
- **`bbl_changes` table size and partitioning**: at institutional scale this table will
  be very large. Three options worth evaluating before the first migration: (1) keep the
  `bigserial` PK and add time-range partitioning — global monotonic order is preserved
  via a shared sequence, old partitions detach to cold storage, but per-entity queries
  scan all partitions without a time filter; (2) drop the sequence and use
  `(rev_id, entity_type, entity_id)` as PK, ordering via `bbl_revs.created_at` —
  freely partitionable, no sequence bottleneck, but "all changes since cursor X" becomes
  a join; (3) keep `bbl_changes` as-is and add a separate lightweight `bbl_events`
  outbox table with a `bigserial` for external consumers (event stream, webhooks) that
  can be pruned after acknowledgement — `bbl_changes` stays large but is only queried
  per-entity. Option 2 is preferred unless an external event stream cursor is needed,
  in which case option 3 adds that without polluting the audit table.
- **Work identifier exclusivity**: enforce unique active DOI ownership at DB level once
  policy is defined?
- **File access control granularity**: per-file (as sketched) or per-work-file-group?
- **Events/conferences as a structured entity**: currently a conference venue can be stored as a free-text field on a work (no schema change needed for basic use). If conferences need deduplication across sources, the same records/identities split used for orgs/persons applies — one flat table vs six. Journal/series editorship is a clean add via `bbl_person_affiliations.kind` or a dedicated link table. Per-work editorship already works via `bbl_work_contributors.role`. The authority model question (flat vs MDM) should be decided before any conference table is added, because it determines the shape of the work-event link and whether a venue-name snapshot is needed on that link row.
