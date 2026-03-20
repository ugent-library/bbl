-- +goose up

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
-- Automated data sources only. Humans are not sources — they are
-- identified by user_id on assertion rows. Source priority is used
-- by auto-pin to select the display value when no human assertion exists.
-- ============================================================

CREATE TABLE bbl_sources (
    id          text PRIMARY KEY,                -- e.g. 'plato', 'orcid', 'wos'
    priority    int NOT NULL DEFAULT 0,           -- higher = more trusted; used by auto-pin
    description text
);

-- ============================================================
-- USERS
-- ============================================================

CREATE TABLE bbl_users (
    id             uuid PRIMARY KEY,
    created_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    username       text NOT NULL UNIQUE,
    email          text NOT NULL COLLATE bbl_case_insensitive,
    name           text NOT NULL,
    role           text NOT NULL,
    deactivate_at  timestamptz,
    person_id      uuid UNIQUE,                  -- FK added below after bbl_people
    auth_providers jsonb NOT NULL DEFAULT '[]',   -- [{"provider":"ugent_oidc"}, ...]
    provenance     jsonb NOT NULL DEFAULT '{}',   -- {"field": {"source": "...", "updated_at": "..."}}
    CHECK (role <> '')
);

CREATE TABLE bbl_user_events (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    kind            text NOT NULL,
    performed_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    payload         jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_user_events (user_id);

CREATE TABLE bbl_user_identifiers (
    user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    source  text NOT NULL REFERENCES bbl_sources (id), -- owner; cleans up its set on each import
    scheme  text NOT NULL,
    val     text NOT NULL,
    PRIMARY KEY (user_id, source, scheme, val),
    UNIQUE (scheme, val), -- each val belongs to at most one user
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE TABLE bbl_user_sources (
    user_id      uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    source       text NOT NULL REFERENCES bbl_sources (id),
    source_id    text NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at   timestamptz,
    PRIMARY KEY (user_id, source),
    UNIQUE (source, source_id)
);

CREATE INDEX ON bbl_user_sources (source, last_seen_at) WHERE expires_at IS NOT NULL;

CREATE INDEX ON bbl_users USING GIN (auth_providers);

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

CREATE TABLE bbl_user_tokens (
    user_id    uuid        NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    provider   text        NOT NULL, -- e.g. 'orcid'
    token      bytea       NOT NULL, -- AES-256-GCM encrypted
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (user_id, provider)
);

-- ============================================================
-- GRANTS
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
    revoked_at    timestamptz,
    note          text,
    CHECK ((scope_type IS NULL) = (scope_id IS NULL)),
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_grants (user_id);
CREATE INDEX ON bbl_grants (scope_type, scope_id);
CREATE INDEX ON bbl_grants (user_id) WHERE revoked_at IS NULL AND expires_at IS NULL;
CREATE INDEX ON bbl_grants (expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================
-- AUDIT: REVS & HISTORY
-- Defined early so assertion tables can FK to bbl_revs.
-- ============================================================

CREATE TABLE bbl_revs (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    user_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    source     text REFERENCES bbl_sources (id)  -- NULL for human revs; both informational
);

CREATE TABLE bbl_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id      bigint NOT NULL REFERENCES bbl_revs (id),
    record_type text NOT NULL,
    record_id   uuid NOT NULL,
    field       text NOT NULL,
    val         jsonb,
    hidden      bool
);

CREATE INDEX ON bbl_history (record_type, record_id);

-- ============================================================
-- ORGANIZATIONS
-- ============================================================

CREATE TABLE bbl_organizations (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind          text NOT NULL,
    status        text NOT NULL DEFAULT 'public',
    start_date    date,
    end_date      date,
    deleted_at    timestamptz,
    deleted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    cache         jsonb NOT NULL DEFAULT '{}',
    CHECK (kind <> ''),
    CHECK (status IN ('public', 'deleted'))
);

CREATE TABLE bbl_organization_sources (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    source          text NOT NULL REFERENCES bbl_sources (id),
    source_id       text NOT NULL,
    record          bytea NOT NULL,
    fetched_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ingested_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (organization_id, source, source_id)
);

CREATE INDEX ON bbl_organization_sources (source, source_id);

CREATE TABLE bbl_organization_assertions (
    id                     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id                 bigint NOT NULL REFERENCES bbl_revs (id),
    organization_id        uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    field                  text NOT NULL,
    val                    jsonb,
    hidden                 bool NOT NULL DEFAULT false,
    organization_source_id uuid REFERENCES bbl_organization_sources (id) ON DELETE CASCADE,
    user_id                uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role                   text,
    asserted_at            timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned                 bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(organization_source_id, user_id) = 1)
);

CREATE INDEX ON bbl_organization_assertions (organization_id, field)
  WHERE pinned = true;

-- Extension table: organization rels need FK columns.
-- Identifiers and names are inlined in bbl_organization_assertions.

CREATE TABLE bbl_organization_assertion_rels (
    assertion_id        bigint PRIMARY KEY REFERENCES bbl_organization_assertions (id) ON DELETE CASCADE,
    rel_organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    kind                text NOT NULL,
    start_date          date,
    end_date            date,
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_organization_assertion_rels (rel_organization_id);

-- ============================================================
-- PEOPLE
-- ============================================================

CREATE TABLE bbl_people (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    status        text NOT NULL DEFAULT 'public',
    deleted_at    timestamptz,
    deleted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    cache         jsonb NOT NULL DEFAULT '{}',
    CHECK (status IN ('public', 'deleted'))
);

ALTER TABLE bbl_users
    ADD CONSTRAINT bbl_users_person_id_fkey
    FOREIGN KEY (person_id)
    REFERENCES bbl_people (id)
    ON DELETE SET NULL;

CREATE TABLE bbl_person_sources (
    id          uuid PRIMARY KEY,
    person_id   uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    source      text NOT NULL REFERENCES bbl_sources (id),
    source_id   text NOT NULL,
    record      bytea NOT NULL,
    fetched_at  timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ingested_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (person_id, source, source_id)
);

CREATE INDEX ON bbl_person_sources (source, source_id);

CREATE TABLE bbl_person_assertions (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id           bigint NOT NULL REFERENCES bbl_revs (id),
    person_id        uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    field            text NOT NULL,
    val              jsonb,
    hidden           bool NOT NULL DEFAULT false,
    person_source_id uuid REFERENCES bbl_person_sources (id) ON DELETE CASCADE,
    user_id          uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role             text,
    asserted_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned           bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(person_source_id, user_id) = 1)
);

CREATE INDEX ON bbl_person_assertions (person_id, field)
  WHERE pinned = true;

-- Extension table: person-organization links need FK columns.
-- Identifiers are inlined in bbl_person_assertions.

CREATE TABLE bbl_person_assertion_organizations (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_person_assertions (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    valid_from      date,
    valid_to        date
);

CREATE INDEX ON bbl_person_assertion_organizations (organization_id);

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
    status        text NOT NULL DEFAULT 'public',
    start_date    date,
    end_date      date,
    deleted_at    timestamptz,
    deleted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    cache         jsonb NOT NULL DEFAULT '{}',
    CHECK (status IN ('public', 'deleted'))
);

CREATE TABLE bbl_project_sources (
    id          uuid PRIMARY KEY,
    project_id  uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    source      text NOT NULL REFERENCES bbl_sources (id),
    source_id   text NOT NULL,
    record      bytea NOT NULL,
    fetched_at  timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ingested_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (project_id, source, source_id)
);

CREATE INDEX ON bbl_project_sources (source, source_id);

CREATE TABLE bbl_project_assertions (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id            bigint NOT NULL REFERENCES bbl_revs (id),
    project_id        uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    field             text NOT NULL,
    val               jsonb,
    hidden            bool NOT NULL DEFAULT false,
    project_source_id uuid REFERENCES bbl_project_sources (id) ON DELETE CASCADE,
    user_id           uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role              text,
    asserted_at       timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned            bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(project_source_id, user_id) = 1)
);

CREATE INDEX ON bbl_project_assertions (project_id, field)
  WHERE pinned = true;

-- Extension table: project-person links need FK columns.
-- Titles, descriptions, identifiers are inlined in bbl_project_assertions.

CREATE TABLE bbl_project_assertion_people (
    assertion_id bigint PRIMARY KEY REFERENCES bbl_project_assertions (id) ON DELETE CASCADE,
    person_id    uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    role         text,
    CHECK (role <> '')
);

CREATE INDEX ON bbl_project_assertion_people (person_id);

-- ============================================================
-- PERSON CANDIDATES
-- Defined after bbl_revs so decided_rev_id FK is valid.
-- ============================================================

CREATE TABLE bbl_person_candidates (
    id             uuid PRIMARY KEY,
    source         text NOT NULL REFERENCES bbl_sources (id),
    source_id      text NOT NULL,
    status         text NOT NULL DEFAULT 'pending', -- pending | accepted | rejected
    confidence     numeric,
    attrs          jsonb NOT NULL DEFAULT '{}',
    fetched_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at     timestamptz,
    decided_at     timestamptz,
    decided_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id bigint REFERENCES bbl_revs (id) ON DELETE SET NULL,
    person_id      uuid REFERENCES bbl_people (id) ON DELETE SET NULL,
    UNIQUE (source, source_id),
    CHECK (status <> '')
);

CREATE INDEX ON bbl_person_candidates (status);
CREATE INDEX ON bbl_person_candidates (source, status);
CREATE INDEX ON bbl_person_candidates (expires_at) WHERE status = 'pending';
CREATE INDEX ON bbl_person_candidates (person_id) WHERE person_id IS NOT NULL;

CREATE TABLE bbl_person_candidate_identifiers (
    candidate_id uuid NOT NULL REFERENCES bbl_person_candidates (id) ON DELETE CASCADE,
    scheme       text NOT NULL,
    val          text NOT NULL,
    PRIMARY KEY (candidate_id, scheme),
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_person_candidate_identifiers (scheme, val);

CREATE TABLE bbl_person_candidate_scores (
    candidate_id uuid NOT NULL REFERENCES bbl_person_candidates (id) ON DELETE CASCADE,
    signal       text NOT NULL,
    score        numeric NOT NULL,
    weight       numeric NOT NULL,
    PRIMARY KEY (candidate_id, signal)
);

-- ============================================================
-- WORKS
-- ============================================================

CREATE TABLE bbl_works (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind          text NOT NULL,
    status        text NOT NULL,       -- 'private' | 'restricted' | 'public' | 'deleted'
    review_status text,                -- NULL | 'pending' | 'in_review' | 'returned'
    delete_kind   text,                -- 'withdrawn' | 'retracted' | 'takedown'
    deleted_at    timestamptz,
    deleted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    cache         jsonb NOT NULL DEFAULT '{}',
    CHECK (kind <> ''),
    CHECK (status <> ''),
    CHECK (review_status <> ''),
    CHECK (delete_kind <> '')
);

CREATE INDEX ON bbl_works (status);
CREATE INDEX ON bbl_works (review_status) WHERE review_status IS NOT NULL;

CREATE TABLE bbl_work_takedowns (
    id                  uuid PRIMARY KEY,
    work_id             uuid NOT NULL REFERENCES bbl_works (id),
    legal_basis         text NOT NULL,
    reference           text,
    requested_at        timestamptz NOT NULL,
    requested_by        text,
    decided_at          timestamptz,
    decided_by_id       uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs_purged_at     timestamptz,
    revs_purged_at      timestamptz,
    notes               text
);

CREATE INDEX ON bbl_work_takedowns (work_id);

-- Work candidates: thin staging table. Defined before bbl_work_sources so
-- candidate_id can be a proper FK.
CREATE TABLE bbl_work_candidates (
    id             uuid PRIMARY KEY,
    source         text NOT NULL REFERENCES bbl_sources (id),
    source_id      text NOT NULL,
    status         text NOT NULL DEFAULT 'pending', -- pending | accepted | rejected
    confidence     numeric,
    attrs          jsonb NOT NULL DEFAULT '{}',
    fetched_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at     timestamptz,
    decided_at     timestamptz,
    decided_by_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    decided_rev_id bigint REFERENCES bbl_revs (id) ON DELETE SET NULL,
    work_id        uuid REFERENCES bbl_works (id) ON DELETE SET NULL,
    UNIQUE (source, source_id),
    CHECK (status <> '')
);

CREATE INDEX ON bbl_work_candidates (status);
CREATE INDEX ON bbl_work_candidates (source, status);
CREATE INDEX ON bbl_work_candidates (expires_at) WHERE status = 'pending';
CREATE INDEX ON bbl_work_candidates (work_id) WHERE work_id IS NOT NULL;

CREATE TABLE bbl_work_candidate_identifiers (
    candidate_id uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    scheme       text NOT NULL,
    val          text NOT NULL,
    PRIMARY KEY (candidate_id, scheme),
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_candidate_identifiers (scheme, val);

CREATE TABLE bbl_work_candidate_people (
    candidate_id uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    person_id    uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    confidence   numeric NOT NULL,
    match_signal text,
    PRIMARY KEY (candidate_id, person_id)
);

CREATE INDEX ON bbl_work_candidate_people (person_id);
CREATE INDEX ON bbl_work_candidate_people (person_id, confidence);

CREATE TABLE bbl_work_candidate_organizations (
    candidate_id    uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    confidence      numeric NOT NULL,
    match_signal    text,
    PRIMARY KEY (candidate_id, organization_id)
);

CREATE INDEX ON bbl_work_candidate_organizations (organization_id);

CREATE TABLE bbl_work_sources (
    id           uuid PRIMARY KEY,
    work_id      uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    source       text NOT NULL REFERENCES bbl_sources (id),
    source_id    text NOT NULL,
    candidate_id uuid REFERENCES bbl_work_candidates (id) ON DELETE SET NULL,
    record       bytea NOT NULL,
    fetched_at   timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ingested_at  timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (work_id, source, source_id)
);

CREATE INDEX ON bbl_work_sources (source, source_id);
CREATE INDEX ON bbl_work_sources (candidate_id) WHERE candidate_id IS NOT NULL;

-- Assertions table: tracks who asserted what about which field.
-- Scalar values and collection items are inlined (val jsonb).
-- FK-bearing items have a thin extension table for the FK columns.

CREATE TABLE bbl_work_assertions (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rev_id         bigint NOT NULL REFERENCES bbl_revs (id),
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    field          text NOT NULL,
    val            jsonb,
    hidden         bool NOT NULL DEFAULT false,
    work_source_id uuid REFERENCES bbl_work_sources (id) ON DELETE CASCADE,
    user_id        uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role           text,
    asserted_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned         bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(work_source_id, user_id) = 1)
);

CREATE INDEX ON bbl_work_assertions (work_id, field)
  WHERE pinned = true;

-- Extension tables: only for collection items that need FK columns.
-- Pure-value collectives (identifiers, classifications, titles, abstracts,
-- lay summaries, notes, keywords) are inlined in bbl_work_assertions.

CREATE TABLE bbl_work_assertion_contributors (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    person_id       uuid REFERENCES bbl_people (id) ON DELETE SET NULL,
    organization_id uuid REFERENCES bbl_organizations (id) ON DELETE SET NULL
);

CREATE INDEX ON bbl_work_assertion_contributors (person_id) WHERE person_id IS NOT NULL;
CREATE INDEX ON bbl_work_assertion_contributors (organization_id) WHERE organization_id IS NOT NULL;

CREATE TABLE bbl_work_assertion_projects (
    assertion_id bigint PRIMARY KEY REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    project_id   uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE RESTRICT
);

CREATE INDEX ON bbl_work_assertion_projects (project_id);

CREATE TABLE bbl_work_assertion_organizations (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE RESTRICT
);

CREATE INDEX ON bbl_work_assertion_organizations (organization_id);

CREATE TABLE bbl_work_assertion_rels (
    assertion_id    bigint PRIMARY KEY REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    related_work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    kind            text NOT NULL,
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_work_assertion_rels (related_work_id);

CREATE TABLE bbl_work_files (
    work_id             uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq                 int NOT NULL,
    object_id           uuid NOT NULL,
    name                text NOT NULL,
    content_type        text NOT NULL,
    size                int NOT NULL,
    sha256              text,
    upload_status       text NOT NULL DEFAULT 'pending',
    access_kind         text NOT NULL DEFAULT 'open',
    embargo_until       timestamptz,
    embargo_access_kind text,
    embargo_lifted_at   timestamptz,
    PRIMARY KEY (work_id, seq),
    CHECK (upload_status <> ''),
    CHECK (access_kind <> ''),
    CHECK (embargo_access_kind <> '')
);

CREATE TABLE bbl_work_reviews (
    id         uuid PRIMARY KEY,
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq        int NOT NULL,
    rev_id     bigint REFERENCES bbl_revs (id) ON DELETE SET NULL,
    user_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind       text NOT NULL,
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (work_id, seq),
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_work_reviews (work_id);

-- ============================================================
-- REPRESENTATIONS & COLLECTIONS
-- ============================================================

CREATE TABLE bbl_work_collections (
    id          uuid PRIMARY KEY,
    name        text NOT NULL UNIQUE,
    description text
);

CREATE TABLE bbl_work_representations (
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme        text NOT NULL,
    record        bytea NOT NULL,
    record_sha256 bytea NOT NULL,
    work_version  int NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (work_id, scheme)
);

CREATE INDEX ON bbl_work_representations (updated_at);

CREATE TABLE bbl_work_collection_works (
    collection_id uuid NOT NULL REFERENCES bbl_work_collections (id) ON DELETE CASCADE,
    work_id       uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    pos           text NOT NULL COLLATE "C",
    PRIMARY KEY (collection_id, work_id),
    UNIQUE (collection_id, pos)
);

CREATE INDEX ON bbl_work_collection_works (work_id);

-- ============================================================
-- LISTS & SUBSCRIPTIONS
-- ============================================================

CREATE TABLE bbl_lists (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    name          text NOT NULL,
    public        boolean NOT NULL DEFAULT false,
    entity_type   text,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL
);

CREATE INDEX ON bbl_lists (created_by_id);

CREATE TABLE bbl_list_items (
    list_id     uuid NOT NULL REFERENCES bbl_lists (id) ON DELETE CASCADE,
    entity_type text NOT NULL,
    entity_id   uuid NOT NULL,
    pos         text NOT NULL COLLATE "C",
    UNIQUE (list_id, entity_type, entity_id),
    UNIQUE (list_id, pos)
);

CREATE INDEX ON bbl_list_items (entity_type, entity_id);

CREATE TABLE bbl_subscriptions (
    id                uuid PRIMARY KEY,
    user_id           uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    topic             text NOT NULL,
    webhook_url       text,
    webhook_secret    text,
    webhook_headers   jsonb NOT NULL DEFAULT '{}',
    status            text NOT NULL DEFAULT 'active',
    failure_count     int NOT NULL DEFAULT 0,
    last_attempted_at timestamptz,
    last_succeeded_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at        timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CHECK (webhook_url IS NOT NULL OR (webhook_secret IS NULL AND webhook_headers = '{}')),
    CHECK (status <> '')
);

CREATE INDEX ON bbl_subscriptions (user_id);
CREATE INDEX ON bbl_subscriptions (topic) WHERE status = 'active';

-- ============================================================
-- VIEWS
-- ============================================================

-- View: all pinned assertion values for a work, grouped by field.
-- Scalar fields have position IS NULL. Collection items have position set.
-- Contributors join to extension table for person_id/organization_id.

-- +goose down
DROP TABLE IF EXISTS bbl_subscriptions CASCADE;
DROP TABLE IF EXISTS bbl_list_items CASCADE;
DROP TABLE IF EXISTS bbl_lists CASCADE;
DROP TABLE IF EXISTS bbl_work_collection_works CASCADE;
DROP TABLE IF EXISTS bbl_work_representations CASCADE;
DROP TABLE IF EXISTS bbl_work_collections CASCADE;
DROP TABLE IF EXISTS bbl_work_reviews CASCADE;
DROP TABLE IF EXISTS bbl_work_files CASCADE;
DROP TABLE IF EXISTS bbl_work_assertion_rels CASCADE;
DROP TABLE IF EXISTS bbl_work_assertion_projects CASCADE;
DROP TABLE IF EXISTS bbl_work_assertion_organizations CASCADE;
DROP TABLE IF EXISTS bbl_work_assertion_contributors CASCADE;
DROP TABLE IF EXISTS bbl_work_assertions CASCADE;
DROP TABLE IF EXISTS bbl_work_sources CASCADE;
DROP TABLE IF EXISTS bbl_work_candidate_organizations CASCADE;
DROP TABLE IF EXISTS bbl_work_candidate_people CASCADE;
DROP TABLE IF EXISTS bbl_work_candidate_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_work_candidates CASCADE;
DROP TABLE IF EXISTS bbl_work_takedowns CASCADE;
DROP TABLE IF EXISTS bbl_works CASCADE;
DROP TABLE IF EXISTS bbl_person_candidate_scores CASCADE;
DROP TABLE IF EXISTS bbl_person_candidate_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_person_candidates CASCADE;
DROP TABLE IF EXISTS bbl_project_assertion_people CASCADE;
DROP TABLE IF EXISTS bbl_project_assertions CASCADE;
DROP TABLE IF EXISTS bbl_project_sources CASCADE;
DROP TABLE IF EXISTS bbl_projects CASCADE;
DROP TABLE IF EXISTS bbl_person_assertion_organizations CASCADE;
DROP TABLE IF EXISTS bbl_person_assertions CASCADE;
DROP TABLE IF EXISTS bbl_person_sources CASCADE;
DROP TABLE IF EXISTS bbl_people CASCADE;
DROP TABLE IF EXISTS bbl_organization_assertion_rels CASCADE;
DROP TABLE IF EXISTS bbl_organization_assertions CASCADE;
DROP TABLE IF EXISTS bbl_organization_sources CASCADE;
DROP TABLE IF EXISTS bbl_organizations CASCADE;
DROP TABLE IF EXISTS bbl_history CASCADE;
DROP TABLE IF EXISTS bbl_revs CASCADE;
DROP TABLE IF EXISTS bbl_grants CASCADE;
DROP TABLE IF EXISTS bbl_user_tokens CASCADE;
DROP TABLE IF EXISTS bbl_user_proxies CASCADE;
DROP TABLE IF EXISTS bbl_user_sources CASCADE;
DROP TABLE IF EXISTS bbl_user_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_user_events CASCADE;
DROP TABLE IF EXISTS bbl_users CASCADE;
DROP TABLE IF EXISTS bbl_sources CASCADE;
DROP COLLATION IF EXISTS bbl_case_insensitive;
