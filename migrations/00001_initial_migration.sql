-- +goose Up

create table bbl_works (
  id uuid primary key,
  kind text not null,
  sub_kind text,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_works_rels (
  id uuid primary key,
  kind text not null,
  work_id uuid not null references bbl_works (id) on delete cascade,
  rel_work_id uuid not null references bbl_works (id) on delete cascade
);

create table bbl_organizations (
  id uuid primary key,
  kind text not null,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_organizations_rels (
  id uuid primary key,
  kind text not null,
  organization_id uuid not null references bbl_organizations (id) on delete cascade,
  rel_organization_id uuid not null references bbl_organizations (id) on delete cascade
);

create table bbl_projects (
  id uuid primary key,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_people (
  id uuid primary key,
  attrs jsonb not null default '{}',
  created_at timestamptz not null default transaction_timestamp(),
  updated_at timestamptz not null default transaction_timestamp()
);

create table bbl_people_organizations (
  id uuid primary key,
  person_id uuid not null references bbl_people (id) on delete cascade,
  organization_id uuid not null references bbl_organizations (id) on delete cascade
);

create table bbl_works_contributors (
  id uuid primary key,
  kind text not null,
  work_id uuid not null references bbl_works (id) on delete cascade,
  person_id uuid references bbl_people (id) on delete set null,
  seq int not null default 1,
  attrs jsonb not null default '{}'
);

create table bbl_works_organizations (
  id uuid primary key,
  work_id uuid not null references bbl_works (id) on delete cascade,
  organization_id uuid not null references bbl_organizations (id) on delete cascade
);

create table bbl_works_projects (
  id uuid primary key,
  work_id uuid not null references bbl_works (id) on delete cascade,
  project_id uuid not null references bbl_projects (id) on delete cascade
);

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

  constraint bbl_changes_record_id_check check (
    (case when organization_id is null then 0 else 1 end) +
    (case when person_id is null then 0 else 1 end) +
    (case when project_id is null then 0 else 1 end) +
    (case when work_id is null then 0 else 1 end) = 1
  )
);

create index on bbl_changes (organization_id) where organization_id is not null;
create index on bbl_changes (person_id) where person_id is not null;
create index on bbl_changes (project_id) where project_id is not null;
create index on bbl_changes (work_id) where work_id is not null;

-- +goose Down

drop table bbl_changes cascade;
drop table bbl_revs cascade;
drop table bbl_people_organizations cascade;
drop table bbl_works_contributors cascade;
drop table bbl_works_organizations cascade;
drop table bbl_works_projects cascade;
drop table bbl_organizations_rels cascade;
drop table bbl_organizations cascade;
drop table bbl_people cascade;
drop table bbl_projects cascade;
drop table bbl_works_rels cascade;
drop table bbl_works cascade;
