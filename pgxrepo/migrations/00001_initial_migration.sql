-- +goose up

CREATE COLLATION bbl_case_insensitive (
  provider = icu,
  locale = 'und-u-ks-level2',
  deterministic = false
);

CREATE TABLE bbl_users (
  id uuid PRIMARY KEY,
  username text NOT NULL UNIQUE,
  email text NOT NULL UNIQUE COLLATE bbl_case_insensitive,
  name text NOT NULL,
  role text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  deactivate_at timestamptz
);

CREATE TABLE BBL_USER_IDENTIFIERS (
  user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
  idx int NOT NULL,
  scheme text NOT NULL,
  val text NOT NULL,
  PRIMARY KEY (user_id, idx),
  UNIQUE (scheme, val)
);

-- TODO IS this necessary?
CREATE INDEX ON bbl_user_identifiers (user_id);

CREATE TABLE bbl_user_proxies (
  user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
  proxy_user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, proxy_user_id),
  CHECK (user_id <> proxy_user_id)
);

CREATE INDEX ON bbl_user_proxies (user_id); -- TODO probably not needed
CREATE INDEX ON bbl_user_proxies (proxy_user_id);

CREATE TABLE bbl_subscriptions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
  topic text NOT NULL,
  webhook_url text NOT NULL
);

CREATE INDEX ON bbl_subscriptions (user_id);
CREATE INDEX ON bbl_subscriptions (topic);

CREATE TABLE bbl_organizations (
  id uuid PRIMARY KEY,
  version int NOT NULL,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  kind text NOT NULL,
  attrs jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE bbl_organization_identifiers (
  organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  idx int NOT NULL,
  scheme text NOT NULL,
  val text NOT NULL,
  uniq boolean NOT NULL,
  PRIMARY KEY (organization_id, idx)
);

-- TODO are these all necessary?
CREATE INDEX ON bbl_organization_identifiers (organization_id);
CREATE UNIQUE INDEX ON bbl_organization_identifiers (scheme, val) WHERE uniq IS TRUE;
CREATE INDEX ON bbl_organization_identifiers (scheme, val);
CREATE INDEX ON bbl_organization_identifiers (uniq);

CREATE TABLE bbl_organization_rels (
  organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  idx int NOT NULL,
  kind text NOT NULL,
  rel_organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  PRIMARY KEY (organization_id, idx),
  CHECK (organization_id <> rel_organization_id)
);

CREATE INDEX ON bbl_organization_rels (organization_id); -- TODO probably not needed
CREATE INDEX ON bbl_organization_rels (rel_organization_id);

CREATE TABLE bbl_people (
  id uuid PRIMARY KEY,
  version int NOT NULL,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  attrs jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE bbl_person_identifiers (
  person_id uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
  idx int NOT NULL,
  scheme text NOT NULL,
  val text NOT NULL,
  uniq boolean NOT NULL,
  PRIMARY KEY (person_id, idx)
);

-- TODO are these all necessary?
CREATE INDEX ON bbl_person_identifiers (person_id);
CREATE UNIQUE INDEX ON bbl_person_identifiers (scheme, val) WHERE uniq IS TRUE;
CREATE INDEX ON bbl_person_identifiers (scheme, val);
CREATE INDEX ON bbl_person_identifiers (uniq);

CREATE TABLE bbl_person_organizations (
  person_id uuid NOT NULL REFERENCES bbl_people (id) ON DELETE CASCADE,
  idx int NOT NULL,
  organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  PRIMARY KEY (person_id, idx)
);

CREATE INDEX ON bbl_person_organizations (person_id); -- TODO probably not needed
CREATE INDEX ON bbl_person_organizations (organization_id);

CREATE TABLE bbl_projects (
  id uuid PRIMARY KEY,
  version int NOT NULL,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  attrs jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE bbl_project_identifiers (
  project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
  idx int NOT NULL,
  scheme text NOT NULL,
  val text NOT NULL,
  uniq boolean NOT NULL,
  PRIMARY KEY (project_id, idx)
);

-- TODO are these all necessary?
CREATE INDEX ON bbl_project_identifiers (project_id);
CREATE UNIQUE INDEX ON bbl_project_identifiers (scheme, val) WHERE uniq IS TRUE;
CREATE INDEX ON bbl_project_identifiers (scheme, val);
CREATE INDEX ON bbl_project_identifiers (uniq);

CREATE TABLE bbl_works (
  id uuid PRIMARY KEY,
  version int NOT NULL,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  created_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  updated_by_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL,
  kind text NOT NULL,
  subkind text,
  status text NOT NULL,
  attrs jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE bbl_work_permissions (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES bbl_users (id) ON DELETE CASCADE,
  kind text NOT NULL
  -- TODO duration of access
);

CREATE INDEX ON bbl_work_permissions (work_id);

CREATE TABLE bbl_work_identifiers (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  scheme text NOT NULL,
  val text NOT NULL,
  uniq boolean NOT NULL,
  PRIMARY KEY (work_id, idx)
);

-- TODO are these all necessary?
CREATE INDEX ON bbl_work_identifiers (work_id);
CREATE UNIQUE INDEX ON bbl_work_identifiers (scheme, val) WHERE uniq IS TRUE;
CREATE INDEX ON bbl_work_identifiers (scheme, val);
CREATE INDEX ON bbl_work_identifiers (uniq);

CREATE TABLE bbl_work_files (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  object_id uuid NOT NULL,
  name text NOT NULL,
  content_type text NOT NULL,
  size int NOT NULL,
  PRIMARY KEY (work_id, idx)
);

-- TODO IS this necessary?
CREATE INDEX ON bbl_work_files (work_id);

CREATE TABLE bbl_work_representations (
    work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
    scheme text NOT NULL,
    record bytea NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (work_id, scheme)
);

CREATE INDEX ON bbl_work_representations (updated_at);

CREATE TABLE bbl_work_rels (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  kind text NOT NULL,
  rel_work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  PRIMARY KEY (work_id, idx),
  CHECK (work_id <> rel_work_id)
);

CREATE INDEX ON bbl_work_rels (work_id); -- TODO probably not needed
CREATE INDEX ON bbl_work_rels (rel_work_id);

CREATE TABLE bbl_work_contributors (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  person_id uuid REFERENCES bbl_people (id) ON DELETE SET NULL,
  attrs jsonb NOT NULL DEFAULT '{}',
  PRIMARY KEY (work_id, idx)
);

CREATE INDEX ON bbl_work_contributors (work_id); -- TODO probably not needed
CREATE INDEX ON bbl_work_contributors (person_id) WHERE person_id IS NOT NULL;

CREATE TABLE bbl_work_organizations (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  organization_id uuid NOT NULL REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  PRIMARY KEY (work_id, idx)
);

CREATE INDEX ON bbl_work_organizations (work_id); -- TODO probably not needed
CREATE INDEX ON bbl_work_organizations (organization_id);

CREATE TABLE bbl_work_projects (
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  project_id uuid NOT NULL REFERENCES bbl_projects (id) ON DELETE CASCADE,
  PRIMARY KEY (work_id, idx)
);

CREATE INDEX ON bbl_work_projects (work_id); -- TODO probably not needed
CREATE INDEX ON bbl_work_projects (project_id);

CREATE TABLE bbl_sets (
  id uuid PRIMARY KEY,
  name text NOT NULL UNIQUE,
  description text,
  public boolean NOT NULL DEFAULT false
);

CREATE INDEX ON bbl_sets (public);

CREATE TABLE bbl_set_works (
  set_id uuid NOT NULL REFERENCES bbl_sets (id) ON DELETE CASCADE,
  work_id uuid NOT NULL REFERENCES bbl_works (id) ON DELETE CASCADE,
  idx int NOT NULL,
  PRIMARY KEY (set_id, work_id)
);

CREATE INDEX ON bbl_set_works (set_id); -- TODO probably not needed
CREATE INDEX ON bbl_set_works (work_id);

CREATE TABLE bbl_revs (
  id uuid PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
  user_id uuid REFERENCES bbl_users (id) ON DELETE SET NULL
);

CREATE TABLE bbl_changes (
  id bigserial PRIMARY KEY,
  rev_id uuid NOT NULL REFERENCES bbl_revs (id) ON DELETE CASCADE,
  organization_id uuid REFERENCES bbl_organizations (id) ON DELETE CASCADE,
  person_id uuid REFERENCES bbl_people (id) ON DELETE CASCADE,
  project_id uuid REFERENCES bbl_projects (id) ON DELETE CASCADE,
  work_id uuid REFERENCES bbl_works (id) ON DELETE CASCADE,
  diff jsonb NOT NULL,

  CHECK (
    (case when organization_id IS NULL THEN 0 ELSE 1 end) +
    (case when person_id IS NULL THEN 0 ELSE 1 end) +
    (case when project_id IS NULL THEN 0 ELSE 1 end) +
    (case when work_id IS NULL THEN 0 ELSE 1 end) = 1
  )
);

CREATE INDEX ON bbl_changes (rev_id);
CREATE INDEX ON bbl_changes (organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX ON bbl_changes (person_id) WHERE person_id IS NOT NULL;
CREATE INDEX ON bbl_changes (project_id) WHERE project_id IS NOT NULL;
CREATE INDEX ON bbl_changes (work_id) WHERE work_id IS NOT NULL;

-- +goose down

DROP TABLE bbl_changes CASCADE;
DROP TABLE bbl_revs CASCADE;
DROP TABLE bbl_set_works CASCADE;
DROP TABLE bbl_sets CASCADE;
DROP TABLE bbl_work_files CASCADE;
DROP TABLE bbl_work_contributors CASCADE;
DROP TABLE bbl_work_organizations CASCADE;
DROP TABLE bbl_work_projects CASCADE;
DROP TABLE bbl_person_organizations CASCADE;
DROP TABLE bbl_organization_identifiers CASCADE;
DROP TABLE bbl_organization_rels CASCADE;
DROP TABLE bbl_organizations CASCADE;
DROP TABLE bbl_person_identifiers CASCADE;
DROP TABLE bbl_people CASCADE;
DROP TABLE bbl_project_identifiers CASCADE;
DROP TABLE bbl_projects CASCADE;
DROP TABLE bbl_work_permissions CASCADE;
DROP TABLE bbl_work_representations CASCADE;
DROP TABLE bbl_work_identifiers CASCADE;
DROP TABLE bbl_work_rels CASCADE;
DROP TABLE bbl_works CASCADE;
DROP TABLE bbl_subscriptions CASCADE;
DROP TABLE bbl_user_proxies CASCADE;
DROP TABLE bbl_user_identifiers CASCADE;
DROP TABLE bbl_users CASCADE;
DROP COLLATION bbl_case_insensitive;