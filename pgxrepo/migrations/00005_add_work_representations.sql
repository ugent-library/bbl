-- +goose up

create table bbl_work_representations (
    work_id uuid not null references bbl_works (id) on delete cascade,
    scheme text not null,
    record bytea not null,
    updated_at timestamptz not null default transaction_timestamp(),
    primary key (work_id, scheme)
);

create index on bbl_work_representations (updated_at);

-- +goose down

drop table bbl_work_representations cascade;