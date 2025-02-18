-- +goose Up

create extension if not exists ltree;
create extension if not exists btree_gist;

create table bbl_recs (
    id uuid primary key,
    kind ltree not null
);

create table bbl_attrs (
    rec_id uuid not null references bbl_recs (id) on delete cascade,
    id uuid primary key,
    kind ltree not null,
    seq int not null default 1,
    val jsonb not null default '{}',
    rel_id uuid references bbl_recs (id),
    unique (rec_id, kind, seq)
);

create table bbl_revs (
    id uuid primary key,
    ts timestamptz not null default transaction_timestamp()
);

create table bbl_changes (
    rev_id uuid not null references bbl_revs (id) on delete cascade,
    rec_id uuid not null,
    op text not null,
    seq int not null default 1,
    args jsonb
);

-- +goose Down

drop table bbl_changes cascade;
drop table bbl_revs cascade;
drop table bbl_attrs cascade;
drop table bbl_recs cascade;
