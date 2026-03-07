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
-- ============================================================

CREATE TABLE bbl_sources (
    id          text PRIMARY KEY,
    label       text NOT NULL,
    priority    int NOT NULL DEFAULT 0,
    description text
);

-- ============================================================
-- USERS
-- ============================================================

CREATE TABLE bbl_users (
    id                 uuid PRIMARY KEY,
    created_at         timestamptz NOT NULL DEFAULT transaction_timestamp(),
    username           text NOT NULL UNIQUE,
    email              text NOT NULL COLLATE bbl_case_insensitive,
    name               text NOT NULL,
    role               text NOT NULL,
    deactivate_at      timestamptz,
    person_identity_id uuid UNIQUE,  -- FK added below after bbl_person_identities
    auth_providers     jsonb NOT NULL DEFAULT '[]'  -- [{"provider":"ugent_oidc"}, ...]
);

CREATE TABLE bbl_user_events (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    performed_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE INDEX ON bbl_user_events (user_id);

CREATE TABLE bbl_user_identifiers (
    user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    source  text NOT NULL REFERENCES bbl_sources (id),  -- owner; cleans up its set on each import
    scheme  text NOT NULL,
    val     text NOT NULL,
    PRIMARY KEY (user_id, source, scheme, val),
    UNIQUE (scheme, val)  -- each val belongs to at most one user
);

CREATE TABLE bbl_user_sources (
    user_id          uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    last_seen_at     timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at       timestamptz,
    PRIMARY KEY (user_id, source),
    UNIQUE (source, source_record_id)
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
    CHECK ((scope_type IS NULL) = (scope_id IS NULL))
);

CREATE INDEX ON bbl_grants (user_id);
CREATE INDEX ON bbl_grants (scope_type, scope_id);
CREATE INDEX ON bbl_grants (user_id) WHERE revoked_at IS NULL AND expires_at IS NULL;
CREATE INDEX ON bbl_grants (expires_at) WHERE expires_at IS NOT NULL;

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
    attrs               jsonb NOT NULL DEFAULT '{}',
    provenance          jsonb NOT NULL DEFAULT '{}',
    attrs_locked_fields text[] NOT NULL DEFAULT '{}'
);

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
-- ============================================================

CREATE TABLE bbl_person_identities (
    id            uuid PRIMARY KEY,
    version       int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs         jsonb NOT NULL DEFAULT '{}',
    provenance    jsonb NOT NULL DEFAULT '{}'
);

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
    id     uuid PRIMARY KEY,
    scheme text NOT NULL,
    value  text NOT NULL,
    UNIQUE (scheme, value)
);

CREATE INDEX ON bbl_person_identifiers (scheme, value);

CREATE TABLE bbl_person_record_identifiers (
    record_id     uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    identifier_id uuid NOT NULL REFERENCES bbl_person_identifiers (id) ON DELETE CASCADE,
    PRIMARY KEY (record_id, identifier_id)
);

CREATE INDEX ON bbl_person_record_identifiers (identifier_id);

CREATE TABLE bbl_person_affiliations (
    id              uuid PRIMARY KEY,
    person_id       uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    role            text,
    valid_from      timestamptz,
    valid_to        timestamptz,
    source          text REFERENCES bbl_sources (id)
);

CREATE INDEX ON bbl_person_affiliations (person_id);
CREATE INDEX ON bbl_person_affiliations (organization_id);
CREATE INDEX ON bbl_person_affiliations (person_id) WHERE valid_to IS NULL;

CREATE TABLE bbl_person_match_candidates (
    id                 uuid PRIMARY KEY,
    record_id_a        uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    record_id_b        uuid NOT NULL REFERENCES bbl_person_records (id) ON DELETE CASCADE,
    status             text NOT NULL DEFAULT 'open',
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
    signal       text NOT NULL,
    score        numeric NOT NULL,
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
    provenance          jsonb NOT NULL DEFAULT '{}',
    attrs_locked_fields text[] NOT NULL DEFAULT '{}'
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

CREATE TABLE bbl_person_project_roles (
    id         uuid PRIMARY KEY,
    person_id  uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    role       text,
    valid_from date,
    valid_to   date,
    source     text REFERENCES bbl_sources (id),
    UNIQUE (person_id, project_id, role, valid_from)
);

CREATE INDEX ON bbl_person_project_roles (person_id);
CREATE INDEX ON bbl_person_project_roles (project_id);
CREATE INDEX ON bbl_person_project_roles (person_id) WHERE valid_to IS NULL;

-- ============================================================
-- AUDIT: REVS
-- Defined before works so bbl_work_sources can FK to bbl_revs.
-- ============================================================

CREATE TABLE bbl_revs (
    id         uuid PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    user_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    source     text REFERENCES bbl_sources (id)
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
    status        text NOT NULL,        -- 'private' | 'public' | 'deleted'
    review_status text,                 -- NULL | 'pending' | 'in_review' | 'returned'
    delete_kind   text,                 -- 'withdrawn' | 'retracted' | 'takedown'
    deleted_at    timestamptz,
    deleted_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    attrs               jsonb NOT NULL DEFAULT '{}',
    provenance          jsonb NOT NULL DEFAULT '{}',
    attrs_locked_fields text[] NOT NULL DEFAULT '{}',
    doc                 jsonb NOT NULL DEFAULT '{}'
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
    mutations_purged_at timestamptz,
    notes               text
);

CREATE INDEX ON bbl_work_takedowns (work_id);

-- Work candidates: thin staging table. Defined before bbl_work_sources so
-- candidate_id can be a proper FK. bbl_work_candidates.work_id back-references
-- bbl_works, completing the cycle without any deferred constraints.
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

CREATE TABLE bbl_work_candidate_identifiers (
    candidate_id uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    scheme       text NOT NULL,
    value        text NOT NULL,
    PRIMARY KEY (candidate_id, scheme)
);

CREATE INDEX ON bbl_work_candidate_identifiers (scheme, value);

CREATE TABLE bbl_work_candidate_persons (
    candidate_id       uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    person_identity_id uuid NOT NULL REFERENCES bbl_person_identities (id) ON DELETE CASCADE,
    confidence         numeric NOT NULL,
    match_signal       text,
    PRIMARY KEY (candidate_id, person_identity_id)
);

CREATE INDEX ON bbl_work_candidate_persons (person_identity_id);
CREATE INDEX ON bbl_work_candidate_persons (person_identity_id, confidence);

CREATE TABLE bbl_work_candidate_organizations (
    candidate_id    uuid NOT NULL REFERENCES bbl_work_candidates (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    confidence      numeric NOT NULL,
    match_signal    text,
    PRIMARY KEY (candidate_id, organization_id)
);

CREATE INDEX ON bbl_work_candidate_organizations (organization_id);

CREATE TABLE bbl_work_sources (
    work_id          uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    source           text NOT NULL REFERENCES bbl_sources (id),
    source_record_id text NOT NULL,
    candidate_id     uuid REFERENCES bbl_work_candidates (id) ON DELETE SET NULL,
    ingested_at      timestamptz NOT NULL DEFAULT transaction_timestamp(),
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

CREATE TABLE bbl_work_contributors (
    work_id            uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    pos                text NOT NULL COLLATE "C",
    person_identity_id uuid REFERENCES bbl_person_identities (id) ON DELETE SET NULL,
    role               text,
    attrs              jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (work_id, pos)
);

CREATE INDEX ON bbl_work_contributors (person_identity_id) WHERE person_identity_id IS NOT NULL;

CREATE TABLE bbl_work_organizations (
    work_id         uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq             int NOT NULL,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE RESTRICT,
    role            text,
    PRIMARY KEY (work_id, seq),
    UNIQUE (work_id, organization_id)
);

CREATE INDEX ON bbl_work_organizations (organization_id);

CREATE TABLE bbl_work_projects (
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq        int NOT NULL,
    project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE RESTRICT,
    PRIMARY KEY (work_id, seq),
    UNIQUE (work_id, project_id)
);

CREATE INDEX ON bbl_work_projects (project_id);

CREATE TABLE bbl_work_rels (
    work_id     uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq         int NOT NULL,
    kind        text NOT NULL,
    rel_work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    PRIMARY KEY (work_id, seq),
    CHECK (work_id <> rel_work_id)
);

CREATE INDEX ON bbl_work_rels (rel_work_id);

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
    PRIMARY KEY (work_id, seq)
);

CREATE TABLE bbl_work_review_messages (
    id         uuid PRIMARY KEY,
    work_id    uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    seq        int NOT NULL,
    rev_id     uuid REFERENCES bbl_revs (id) ON DELETE SET NULL,
    author_id  uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    kind       text NOT NULL,
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    UNIQUE (work_id, seq)
);

CREATE INDEX ON bbl_work_review_messages (work_id);

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
-- AUDIT: MUTATIONS
-- ============================================================

CREATE TABLE bbl_mutations (
    id          bigserial PRIMARY KEY,
    rev_id      uuid NOT NULL REFERENCES bbl_revs (id),
    name        text NOT NULL,
    entity_type text NOT NULL,
    entity_id   uuid NOT NULL,
    op_type     text NOT NULL,
    diff        jsonb NOT NULL
);

CREATE INDEX ON bbl_mutations (rev_id);
CREATE INDEX ON bbl_mutations (entity_type, entity_id);

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
    CHECK (webhook_url IS NOT NULL OR (webhook_secret IS NULL AND webhook_headers = '{}'))
);

CREATE INDEX ON bbl_subscriptions (user_id);
CREATE INDEX ON bbl_subscriptions (topic) WHERE status = 'active';

-- +goose down

DROP TABLE bbl_subscriptions CASCADE;
DROP TABLE bbl_list_items CASCADE;
DROP TABLE bbl_lists CASCADE;
DROP TABLE bbl_mutations CASCADE;
DROP TABLE bbl_work_collection_works CASCADE;
DROP TABLE bbl_work_representations CASCADE;
DROP TABLE bbl_work_collections CASCADE;
DROP TABLE bbl_work_review_messages CASCADE;
DROP TABLE bbl_work_files CASCADE;
DROP TABLE bbl_work_rels CASCADE;
DROP TABLE bbl_work_projects CASCADE;
DROP TABLE bbl_work_organizations CASCADE;
DROP TABLE bbl_work_contributors CASCADE;
DROP TABLE bbl_work_identifiers CASCADE;
DROP TABLE bbl_work_sources CASCADE;
DROP TABLE bbl_work_candidate_organizations CASCADE;
DROP TABLE bbl_work_candidate_persons CASCADE;
DROP TABLE bbl_work_candidate_identifiers CASCADE;
DROP TABLE bbl_work_candidates CASCADE;
DROP TABLE bbl_work_takedowns CASCADE;
DROP TABLE bbl_works CASCADE;
DROP TABLE bbl_revs CASCADE;
DROP TABLE bbl_person_project_roles CASCADE;
DROP TABLE bbl_project_sources CASCADE;
DROP TABLE bbl_project_identifiers CASCADE;
DROP TABLE bbl_projects CASCADE;
DROP TABLE bbl_person_match_scores CASCADE;
DROP TABLE bbl_person_match_candidates CASCADE;
DROP TABLE bbl_person_affiliations CASCADE;
DROP TABLE bbl_person_record_identifiers CASCADE;
DROP TABLE bbl_person_identifiers CASCADE;
DROP TABLE bbl_person_identity_records CASCADE;
DROP TABLE bbl_person_records CASCADE;
DROP TABLE bbl_person_identities CASCADE;
DROP TABLE bbl_organization_sources CASCADE;
DROP TABLE bbl_organization_rels CASCADE;
DROP TABLE bbl_organization_identifiers CASCADE;
DROP TABLE bbl_organizations CASCADE;
DROP TABLE bbl_grants CASCADE;
DROP TABLE bbl_user_proxies CASCADE;
DROP TABLE bbl_user_sources CASCADE;
DROP TABLE bbl_user_identifiers CASCADE;
DROP TABLE bbl_user_events CASCADE;
DROP TABLE bbl_users CASCADE;
DROP TABLE bbl_sources CASCADE;
DROP COLLATION bbl_case_insensitive;
