-- +goose up

create extension if not exists citext; -- TODO use collation instead

create table bbl_users (
  id uuid primary key,
  username text not null unique,
  email citext not null unique,
  name text not null,
  role text not null,
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp(),
  deactivate_at timestamptz
);

create table bbl_user_identifiers (
  user_id uuid not null references bbl_users (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  primary key (user_id, idx),
  unique (scheme, val)
);

-- TODO is this necessary?
create index on bbl_user_identifiers (user_id);

create table bbl_user_proxies (
  user_id uuid not null references bbl_users (id) on delete cascade,
  proxy_user_id uuid not null references bbl_users (id) on delete cascade,
  primary key (user_id, proxy_user_id),
  check (user_id <> proxy_user_id)
);

create index on bbl_user_proxies (user_id); -- TODO probably not needed
create index on bbl_user_proxies (proxy_user_id);

create table bbl_subscriptions (
  id uuid primary key,
  user_id uuid not null references bbl_users (id) on delete cascade,
  topic text not null,
  webhook_url text not null
);

create index on bbl_subscriptions (user_id);
create index on bbl_subscriptions (topic);

create table bbl_organizations (
  id uuid primary key,
  version int not null,
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp(),
  created_by_id uuid references bbl_users (id) on delete set null,
  updated_by_id uuid references bbl_users (id) on delete set null,
  kind text not null,
  attrs jsonb not null default '{}'
);

create table bbl_organization_identifiers (
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (organization_id, idx)
);

-- TODO are these all necessary?
create index on bbl_organization_identifiers (organization_id);
create unique index on bbl_organization_identifiers (scheme, val) where uniq is true;
create index on bbl_organization_identifiers (scheme, val);
create index on bbl_organization_identifiers (uniq);

create table bbl_organization_rels (
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  idx int not null,
  kind text not null,
  rel_organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (organization_id, idx),
  check (organization_id <> rel_organization_id)
);

create index on bbl_organization_rels (organization_id); -- TODO probably not needed
create index on bbl_organization_rels (rel_organization_id);

create table bbl_people (
  id uuid primary key,
  version int not null,
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp(),
  created_by_id uuid references bbl_users (id) on delete set null,
  updated_by_id uuid references bbl_users (id) on delete set null,
  attrs jsonb not null default '{}'
);

create table bbl_person_identifiers (
  person_id uuid not null references bbl_people (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (person_id, idx)
);

-- TODO are these all necessary?
create index on bbl_person_identifiers (person_id);
create unique index on bbl_person_identifiers (scheme, val) where uniq is true;
create index on bbl_person_identifiers (scheme, val);
create index on bbl_person_identifiers (uniq);

create table bbl_person_organizations (
  person_id uuid not null references bbl_people (id) on delete cascade,
  idx int not null,
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (person_id, idx)
);

create index on bbl_person_organizations (person_id); -- TODO probably not needed
create index on bbl_person_organizations (organization_id);

create table bbl_projects (
  id uuid primary key,
  version int not null,
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp(),
  created_by_id uuid references bbl_users (id) on delete set null,
  updated_by_id uuid references bbl_users (id) on delete set null,
  attrs jsonb not null default '{}'
);

create table bbl_project_identifiers (
  project_id uuid not null references bbl_projects (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (project_id, idx)
);

-- TODO are these all necessary?
create index on bbl_project_identifiers (project_id);
create unique index on bbl_project_identifiers (scheme, val) where uniq is true;
create index on bbl_project_identifiers (scheme, val);
create index on bbl_project_identifiers (uniq);

create table bbl_works (
  id uuid primary key,
  version int not null,
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp(),
  created_by_id uuid references bbl_users (id) on delete set null,
  updated_by_id uuid references bbl_users (id) on delete set null,
  kind text not null,
  subkind text,
  status text not null,
  attrs jsonb not null default '{}'
);

create table bbl_work_permissions (
  work_id uuid not null references bbl_works (id) on delete cascade,
  user_id uuid not null references bbl_users (id) on delete cascade,
  kind text not null
  -- TODO duration of access
);

create index on bbl_work_permissions (work_id);

create table bbl_work_identifiers (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (work_id, idx)
);

-- TODO are these all necessary?
create index on bbl_work_identifiers (work_id);
create unique index on bbl_work_identifiers (scheme, val) where uniq is true;
create index on bbl_work_identifiers (scheme, val);
create index on bbl_work_identifiers (uniq);

create table bbl_work_files (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  object_id uuid not null,
  name text not null,
  content_type text not null,
  size int not null,
  primary key (work_id, idx)
);

-- TODO is this necessary?
create index on bbl_work_files (work_id);

create table bbl_work_representations (
    work_id uuid not null references bbl_works (id) on delete cascade,
    scheme text not null,
    record bytea not null,
    updated_at timestamptz not null default transaction_timestamp(),
    primary key (work_id, scheme)
);

create index on bbl_work_representations (updated_at);

create table bbl_work_rels (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  kind text not null,
  rel_work_id uuid not null references bbl_works (id) on delete cascade,
  primary key (work_id, idx),
  check (work_id <> rel_work_id)
);

create index on bbl_work_rels (work_id); -- TODO probably not needed
create index on bbl_work_rels (rel_work_id);

create table bbl_work_contributors (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  person_id uuid references bbl_people (id) on delete set null,
  attrs jsonb not null default '{}',
  primary key (work_id, idx)
);

create index on bbl_work_contributors (work_id); -- TODO probably not needed
create index on bbl_work_contributors (person_id) where person_id is not null;

create table bbl_work_organizations (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (work_id, idx)
);

create index on bbl_work_organizations (work_id); -- TODO probably not needed
create index on bbl_work_organizations (organization_id);

create table bbl_work_projects (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  project_id uuid not null references bbl_projects (id) on delete cascade,
  primary key (work_id, idx)
);

create index on bbl_work_projects (work_id); -- TODO probably not needed
create index on bbl_work_projects (project_id);

create table bbl_sets (
  id uuid primary key,
  name text not null unique,
  description text,
  public boolean not null default false
);

create index on bbl_sets (public);

create table bbl_set_works (
  set_id uuid not null references bbl_sets (id) on delete cascade,
  work_id uuid not null references bbl_works (id) on delete cascade,
  primary key (set_id, work_id)
);

create index on bbl_set_works (set_id); -- TODO probably not needed
create index on bbl_set_works (work_id);

create table bbl_revs (
  id uuid primary key,
  created_at timestamptz not null default transaction_timestamp(),
  user_id uuid references bbl_users (id) on delete set null
);

create table bbl_changes (
  id bigserial primary key,
  rev_id uuid not null references bbl_revs (id) on delete cascade,
  organization_id uuid references bbl_organizations (id) on delete cascade,
  person_id uuid references bbl_people (id) on delete cascade,
  project_id uuid references bbl_projects (id) on delete cascade,
  work_id uuid references bbl_works (id) on delete cascade,
  diff jsonb not null,

  check (
    (case when organization_id is null then 0 else 1 end) +
    (case when person_id is null then 0 else 1 end) +
    (case when project_id is null then 0 else 1 end) +
    (case when work_id is null then 0 else 1 end) = 1
  )
);

create index on bbl_changes (rev_id);
create index on bbl_changes (organization_id) where organization_id is not null;
create index on bbl_changes (person_id) where person_id is not null;
create index on bbl_changes (project_id) where project_id is not null;
create index on bbl_changes (work_id) where work_id is not null;

-- +goose down

drop table bbl_changes cascade;
drop table bbl_revs cascade;
drop table bbl_set_works cascade;
drop table bbl_sets cascade;
drop table bbl_work_files cascade;
drop table bbl_work_contributors cascade;
drop table bbl_work_organizations cascade;
drop table bbl_work_projects cascade;
drop table bbl_person_organizations cascade;
drop table bbl_organization_identifiers cascade;
drop table bbl_organization_rels cascade;
drop table bbl_organizations cascade;
drop table bbl_person_identifiers cascade;
drop table bbl_people cascade;
drop table bbl_project_identifiers cascade;
drop table bbl_projects cascade;
drop table bbl_work_permissions cascade;
drop table bbl_work_representations cascade;
drop table bbl_work_identifiers cascade;
drop table bbl_work_rels cascade;
drop table bbl_works cascade;
drop table bbl_subscriptions cascade;
drop table bbl_user_proxies cascade;
drop table bbl_user_identifiers cascade;
drop table bbl_users cascade;
