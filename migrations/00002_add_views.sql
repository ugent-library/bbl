-- +goose Up

create view bbl_organizations_view as
select o.id,
       o.kind,
       o.attrs,
       json_agg(distinct jsonb_build_object('id', o_r.id, 'kind', o_r.kind, 'organization_id', o_r.rel_organization_id)) filter (where o_r.id is not null) as rels,
       o.created_at,
       o.updated_at
from bbl_organizations o
left join bbl_organizations_rels o_r on o.id = o_r.organization_id
group by o.id;

create view bbl_works_view as
select w.id,
       w.kind,
       w.sub_kind,
       w.attrs,
       json_agg(distinct jsonb_build_object('id', w_r.id, 'kind', w_r.kind, 'work_id', w_r.rel_work_id)) filter (where w_r.id is not null) as rels,
       w.created_at,
       w.updated_at
from bbl_works w
left join bbl_works_rels w_r on w.id = w_r.work_id
group by w.id;

-- +goose Down

drop view bbl_organizations_view;
drop view bbl_works_view;