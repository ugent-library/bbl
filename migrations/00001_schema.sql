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
-- AUDIT: REVS & ASSERTION SEQUENCE
-- Defined early so assertion tables can FK to bbl_revs.
-- ============================================================

CREATE TABLE bbl_revs (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    user_id    uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    source     text REFERENCES bbl_sources (id)  -- NULL for human revs; both informational
);

CREATE SEQUENCE bbl_assertion_seq;

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
    id                     bigint PRIMARY KEY DEFAULT nextval('bbl_assertion_seq'),
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

CREATE UNIQUE INDEX ON bbl_organization_assertions (organization_id, field, organization_source_id)
  WHERE organization_source_id IS NOT NULL;
CREATE UNIQUE INDEX ON bbl_organization_assertions (organization_id, field)
  WHERE pinned = true;

CREATE TABLE bbl_organization_identifiers (
    id                     uuid PRIMARY KEY,
    assertion_id           bigint NOT NULL REFERENCES bbl_organization_assertions (id) ON DELETE CASCADE,
    organization_id        uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    scheme                 text NOT NULL,
    val                    text NOT NULL,
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_organization_identifiers (organization_id);
CREATE INDEX ON bbl_organization_identifiers (scheme, val);

CREATE TABLE bbl_organization_names (
    id                     uuid PRIMARY KEY,
    assertion_id           bigint NOT NULL REFERENCES bbl_organization_assertions (id) ON DELETE CASCADE,
    organization_id        uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    lang                   text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val                    text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_organization_names (organization_id);

CREATE TABLE bbl_organization_rels (
    id                     uuid PRIMARY KEY,
    assertion_id           bigint NOT NULL REFERENCES bbl_organization_assertions (id) ON DELETE CASCADE,
    organization_id        uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    rel_organization_id    uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    kind                   text NOT NULL,
    start_date             date,
    end_date               date,
    CHECK (organization_id <> rel_organization_id),
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_organization_rels (organization_id);
CREATE INDEX ON bbl_organization_rels (rel_organization_id);
CREATE INDEX ON bbl_organization_rels (organization_id) WHERE end_date IS NULL;

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
    id               bigint PRIMARY KEY DEFAULT nextval('bbl_assertion_seq'),
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

CREATE UNIQUE INDEX ON bbl_person_assertions (person_id, field, person_source_id)
  WHERE person_source_id IS NOT NULL;
CREATE UNIQUE INDEX ON bbl_person_assertions (person_id, field)
  WHERE pinned = true;

CREATE TABLE bbl_person_identifiers (
    id               uuid PRIMARY KEY,
    assertion_id     bigint NOT NULL REFERENCES bbl_person_assertions (id) ON DELETE CASCADE,
    person_id        uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    scheme           text NOT NULL,
    val              text NOT NULL,
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_person_identifiers (person_id);
CREATE INDEX ON bbl_person_identifiers (scheme, val);

CREATE TABLE bbl_person_organizations (
    id               uuid PRIMARY KEY,
    assertion_id     bigint NOT NULL REFERENCES bbl_person_assertions (id) ON DELETE CASCADE,
    person_id        uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    organization_id  uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
    valid_from       date,
    valid_to         date
);

CREATE INDEX ON bbl_person_organizations (person_id);
CREATE INDEX ON bbl_person_organizations (organization_id);

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
    id                bigint PRIMARY KEY DEFAULT nextval('bbl_assertion_seq'),
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

CREATE UNIQUE INDEX ON bbl_project_assertions (project_id, field, project_source_id)
  WHERE project_source_id IS NOT NULL;
CREATE UNIQUE INDEX ON bbl_project_assertions (project_id, field)
  WHERE pinned = true;

CREATE TABLE bbl_project_titles (
    id                uuid PRIMARY KEY,
    assertion_id      bigint NOT NULL REFERENCES bbl_project_assertions (id) ON DELETE CASCADE,
    project_id        uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    lang              text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val               text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_project_titles (project_id);

CREATE TABLE bbl_project_descriptions (
    id                uuid PRIMARY KEY,
    assertion_id      bigint NOT NULL REFERENCES bbl_project_assertions (id) ON DELETE CASCADE,
    project_id        uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    lang              text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val               text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_project_descriptions (project_id);

CREATE TABLE bbl_project_identifiers (
    id                uuid PRIMARY KEY,
    assertion_id      bigint NOT NULL REFERENCES bbl_project_assertions (id) ON DELETE CASCADE,
    project_id        uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    scheme            text NOT NULL,
    val               text NOT NULL,
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_project_identifiers (project_id);
CREATE INDEX ON bbl_project_identifiers (scheme, val);

CREATE TABLE bbl_project_people (
    id                uuid PRIMARY KEY,
    assertion_id      bigint NOT NULL REFERENCES bbl_project_assertions (id) ON DELETE CASCADE,
    project_id        uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
    person_id         uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
    role              text,
    CHECK (role <> '')
);

CREATE INDEX ON bbl_project_people (project_id);
CREATE INDEX ON bbl_project_people (person_id);

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
-- Scalar values are inlined (val). Collective values live in relation tables with assertion_id FK.

CREATE TABLE bbl_work_assertions (
    id             bigint PRIMARY KEY DEFAULT nextval('bbl_assertion_seq'),
    rev_id         bigint NOT NULL REFERENCES bbl_revs (id),
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    field          text NOT NULL,
    val            jsonb,              -- scalar value; NULL for collective fields
    hidden         bool NOT NULL DEFAULT false,
    work_source_id uuid REFERENCES bbl_work_sources (id) ON DELETE CASCADE,
    user_id        uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
    role           text,
    asserted_at    timestamptz NOT NULL DEFAULT transaction_timestamp(),
    pinned         bool NOT NULL DEFAULT false,
    CHECK (field <> ''),
    CHECK (num_nonnulls(work_source_id, user_id) = 1)
);

CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field, work_source_id)
  WHERE work_source_id IS NOT NULL;
CREATE UNIQUE INDEX ON bbl_work_assertions (work_id, field)
  WHERE pinned = true;

CREATE TABLE bbl_work_identifiers (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme         text NOT NULL,
    val            text NOT NULL,
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_identifiers (work_id);
CREATE INDEX ON bbl_work_identifiers (scheme, val);

CREATE TABLE bbl_work_classifications (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme         text NOT NULL,
    val            text NOT NULL,
    CHECK (scheme <> ''),
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_classifications (work_id);
CREATE INDEX ON bbl_work_classifications (scheme, val);

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

CREATE INDEX ON bbl_work_contributors (work_id);
CREATE INDEX ON bbl_work_contributors (person_id) WHERE person_id IS NOT NULL;
CREATE INDEX ON bbl_work_contributors USING gin (roles) WHERE roles != '{}';

CREATE TABLE bbl_work_titles (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    lang           text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val            text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_titles (work_id);

CREATE TABLE bbl_work_abstracts (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    lang           text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val            text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_abstracts (work_id);

CREATE TABLE bbl_work_lay_summaries (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    lang           text NOT NULL DEFAULT 'und' CHECK (lang <> ''),
    val            text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_lay_summaries (work_id);

CREATE TABLE bbl_work_notes (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    val            text NOT NULL,
    kind           text,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_notes (work_id);

CREATE TABLE bbl_work_keywords (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    val            text NOT NULL,
    CHECK (val <> '')
);

CREATE INDEX ON bbl_work_keywords (work_id);

CREATE TABLE bbl_work_projects (
    id             uuid PRIMARY KEY,
    assertion_id   bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id        uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    project_id     uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE RESTRICT
);

CREATE INDEX ON bbl_work_projects (work_id);
CREATE INDEX ON bbl_work_projects (project_id);

CREATE TABLE bbl_work_organizations (
    id              uuid PRIMARY KEY,
    assertion_id    bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id         uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE RESTRICT
);

CREATE INDEX ON bbl_work_organizations (work_id);
CREATE INDEX ON bbl_work_organizations (organization_id);

CREATE TABLE bbl_work_rels (
    id              uuid PRIMARY KEY,
    assertion_id    bigint NOT NULL REFERENCES bbl_work_assertions (id) ON DELETE CASCADE,
    work_id         uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    related_work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    kind            text NOT NULL,
    CHECK (work_id <> related_work_id),
    CHECK (kind <> '')
);

CREATE INDEX ON bbl_work_rels (work_id);
CREATE INDEX ON bbl_work_rels (related_work_id);

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

CREATE VIEW bbl_works_view AS
SELECT w.*,
       w_sf.str_fields,
       w_i.identifiers,
       w_cl.classifications,
       w_c.contributors,
       w_t.titles,
       w_a.abstracts,
       w_ls.lay_summaries,
       w_n.notes,
       w_kw.keywords
FROM bbl_works w
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('field', sf.field, 'val', sf.val)
    ORDER BY sf.field
  ) FILTER (WHERE sf.work_id IS NOT NULL) AS str_fields
  FROM bbl_work_assertions sf
  WHERE sf.work_id = w.id AND sf.pinned = true AND sf.hidden = false AND sf.val IS NOT NULL
) w_sf ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('scheme', i.scheme, 'val', i.val)
    ORDER BY i.scheme, i.val
  ) FILTER (WHERE i.work_id IS NOT NULL) AS identifiers
  FROM bbl_work_identifiers i
  JOIN bbl_work_assertions a ON a.id = i.assertion_id
  WHERE i.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_i ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('scheme', cl.scheme, 'val', cl.val)
    ORDER BY cl.scheme, cl.val
  ) FILTER (WHERE cl.work_id IS NOT NULL) AS classifications
  FROM bbl_work_classifications cl
  JOIN bbl_work_assertions a ON a.id = cl.assertion_id
  WHERE cl.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_cl ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object(
      'position', c.position,
      'kind', c.kind,
      'person_id', c.person_id,
      'name', c.name,
      'given_name', c.given_name,
      'family_name', c.family_name,
      'roles', c.roles
    ) ORDER BY c.position
  ) FILTER (WHERE c.work_id IS NOT NULL) AS contributors
  FROM bbl_work_contributors c
  JOIN bbl_work_assertions a ON a.id = c.assertion_id
  WHERE c.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_c ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('lang', t.lang, 'val', t.val)
    ORDER BY t.lang, t.val
  ) FILTER (WHERE t.work_id IS NOT NULL) AS titles
  FROM bbl_work_titles t
  JOIN bbl_work_assertions a ON a.id = t.assertion_id
  WHERE t.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_t ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('lang', ab.lang, 'val', ab.val)
    ORDER BY ab.lang
  ) FILTER (WHERE ab.work_id IS NOT NULL) AS abstracts
  FROM bbl_work_abstracts ab
  JOIN bbl_work_assertions a ON a.id = ab.assertion_id
  WHERE ab.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_a ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('lang', ls.lang, 'val', ls.val)
    ORDER BY ls.lang
  ) FILTER (WHERE ls.work_id IS NOT NULL) AS lay_summaries
  FROM bbl_work_lay_summaries ls
  JOIN bbl_work_assertions a ON a.id = ls.assertion_id
  WHERE ls.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_ls ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('kind', n.kind, 'val', n.val)
    ORDER BY n.kind
  ) FILTER (WHERE n.work_id IS NOT NULL) AS notes
  FROM bbl_work_notes n
  JOIN bbl_work_assertions a ON a.id = n.assertion_id
  WHERE n.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_n ON true
LEFT JOIN LATERAL (
  SELECT json_agg(
    json_build_object('val', kw.val)
    ORDER BY kw.val
  ) FILTER (WHERE kw.work_id IS NOT NULL) AS keywords
  FROM bbl_work_keywords kw
  JOIN bbl_work_assertions a ON a.id = kw.assertion_id
  WHERE kw.work_id = w.id AND a.pinned = true AND a.hidden = false
) w_kw ON true;

-- +goose down

DROP VIEW IF EXISTS bbl_works_view;
DROP TABLE IF EXISTS bbl_subscriptions CASCADE;
DROP TABLE IF EXISTS bbl_list_items CASCADE;
DROP TABLE IF EXISTS bbl_lists CASCADE;
DROP TABLE IF EXISTS bbl_work_collection_works CASCADE;
DROP TABLE IF EXISTS bbl_work_representations CASCADE;
DROP TABLE IF EXISTS bbl_work_collections CASCADE;
DROP TABLE IF EXISTS bbl_work_reviews CASCADE;
DROP TABLE IF EXISTS bbl_work_files CASCADE;
DROP TABLE IF EXISTS bbl_work_rels CASCADE;
DROP TABLE IF EXISTS bbl_work_projects CASCADE;
DROP TABLE IF EXISTS bbl_work_organizations CASCADE;
DROP TABLE IF EXISTS bbl_work_keywords CASCADE;
DROP TABLE IF EXISTS bbl_work_notes CASCADE;
DROP TABLE IF EXISTS bbl_work_lay_summaries CASCADE;
DROP TABLE IF EXISTS bbl_work_abstracts CASCADE;
DROP TABLE IF EXISTS bbl_work_titles CASCADE;
DROP TABLE IF EXISTS bbl_work_contributors CASCADE;
DROP TABLE IF EXISTS bbl_work_classifications CASCADE;
DROP TABLE IF EXISTS bbl_work_identifiers CASCADE;
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
DROP TABLE IF EXISTS bbl_revs CASCADE;
DROP SEQUENCE IF EXISTS bbl_assertion_seq CASCADE;
DROP TABLE IF EXISTS bbl_project_people CASCADE;
DROP TABLE IF EXISTS bbl_project_descriptions CASCADE;
DROP TABLE IF EXISTS bbl_project_titles CASCADE;
DROP TABLE IF EXISTS bbl_project_sources CASCADE;
DROP TABLE IF EXISTS bbl_project_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_project_assertions CASCADE;
DROP TABLE IF EXISTS bbl_projects CASCADE;
DROP TABLE IF EXISTS bbl_person_organizations CASCADE;
DROP TABLE IF EXISTS bbl_person_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_person_assertions CASCADE;
DROP TABLE IF EXISTS bbl_person_sources CASCADE;
DROP TABLE IF EXISTS bbl_people CASCADE;
DROP TABLE IF EXISTS bbl_organization_sources CASCADE;
DROP TABLE IF EXISTS bbl_organization_rels CASCADE;
DROP TABLE IF EXISTS bbl_organization_names CASCADE;
DROP TABLE IF EXISTS bbl_organization_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_organization_assertions CASCADE;
DROP TABLE IF EXISTS bbl_organizations CASCADE;
DROP TABLE IF EXISTS bbl_grants CASCADE;
DROP TABLE IF EXISTS bbl_user_tokens CASCADE;
DROP TABLE IF EXISTS bbl_user_proxies CASCADE;
DROP TABLE IF EXISTS bbl_user_sources CASCADE;
DROP TABLE IF EXISTS bbl_user_identifiers CASCADE;
DROP TABLE IF EXISTS bbl_user_events CASCADE;
DROP TABLE IF EXISTS bbl_users CASCADE;
DROP TABLE IF EXISTS bbl_sources CASCADE;
DROP COLLATION IF EXISTS bbl_case_insensitive;
