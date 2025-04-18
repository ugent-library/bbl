-- +goose up

create table bbl_organizations (
  id uuid primary key,
  kind text not null,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_organizations_identifiers (
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (organization_id, idx)
);

-- TODO are these all necessary?
create index on bbl_organizations_identifiers (organization_id);
create unique index on bbl_organizations_identifiers (scheme, val) where uniq is true;
create index on bbl_organizations_identifiers (scheme, val);
create index on bbl_organizations_identifiers (uniq);

create table bbl_organizations_rels (
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  idx int not null,
  kind text not null,
  rel_organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (organization_id, idx),
  check (organization_id <> rel_organization_id)
);

create index on bbl_organizations_rels (organization_id); -- TODO probably not needed
create index on bbl_organizations_rels (rel_organization_id);

create table bbl_people (
  id uuid primary key,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_people_identifiers (
  person_id uuid not null references bbl_people (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (person_id, idx)
);

-- TODO are these all necessary?
create index on bbl_people_identifiers (person_id);
create unique index on bbl_people_identifiers (scheme, val) where uniq is true;
create index on bbl_people_identifiers (scheme, val);
create index on bbl_people_identifiers (uniq);

create table bbl_people_organizations (
  person_id uuid not null references bbl_people (id) on delete cascade,
  idx int not null,
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (person_id, idx)
);

create index on bbl_people_organizations (person_id); -- TODO probably not needed
create index on bbl_people_organizations (organization_id);

create table bbl_projects (
  id uuid primary key,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_projects_identifiers (
  project_id uuid not null references bbl_projects (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (project_id, idx)
);

-- TODO are these all necessary?
create index on bbl_projects_identifiers (project_id);
create unique index on bbl_projects_identifiers (scheme, val) where uniq is true;
create index on bbl_projects_identifiers (scheme, val);
create index on bbl_projects_identifiers (uniq);

create table bbl_works (
  id uuid primary key,
  kind text not null,
  sub_kind text,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_works_identifiers (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  scheme text not null,
  val text not null,
  uniq boolean not null,
  primary key (work_id, idx)
);

-- TODO are these all necessary?
create index on bbl_works_identifiers (work_id);
create unique index on bbl_works_identifiers (scheme, val) where uniq is true;
create index on bbl_works_identifiers (scheme, val);
create index on bbl_works_identifiers (uniq);

create table bbl_works_rels (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  kind text not null,
  rel_work_id uuid not null references bbl_works (id) on delete cascade,
  primary key (work_id, idx),
  check (work_id <> rel_work_id)
);

create index on bbl_works_rels (work_id); -- TODO probably not needed
create index on bbl_works_rels (rel_work_id);

create table bbl_works_contributors (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  person_id uuid references bbl_people (id) on delete set null,
  attrs jsonb not null default '{}',
  primary key (work_id, idx)
);

create index on bbl_works_contributors (work_id); -- TODO probably not needed
create index on bbl_works_contributors (person_id) where person_id is not null;

create table bbl_works_organizations (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  primary key (work_id, idx)
);

create index on bbl_works_organizations (work_id); -- TODO probably not needed
create index on bbl_works_organizations (organization_id);

create table bbl_works_projects (
  work_id uuid not null references bbl_works (id) on delete cascade,
  idx int not null,
  project_id uuid not null references bbl_projects (id) on delete cascade,
  primary key (work_id, idx)
);

create index on bbl_works_projects (work_id); -- TODO probably not needed
create index on bbl_works_projects (project_id);

create table bbl_revs (
  id uuid primary key,
  created_at timestamptz default transaction_timestamp()
);

create table bbl_changes (
  id bigint generated always as identity,
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
drop table bbl_people_organizations cascade;
drop table bbl_works_contributors cascade;
drop table bbl_works_organizations cascade;
drop table bbl_works_projects cascade;
drop table bbl_organizations_identifiers cascade;
drop table bbl_organizations_rels cascade;
drop table bbl_organizations cascade;
drop table bbl_people_identifiers cascade;
drop table bbl_people cascade;
drop table bbl_projects_identifiers cascade;
drop table bbl_projects cascade;
drop table bbl_works_identifiers cascade;
drop table bbl_works_rels cascade;
drop table bbl_works cascade;
