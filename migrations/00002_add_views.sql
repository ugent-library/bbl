-- +goose Up

create view bbl_organizations_view as
select o.*,
       o_r.rels as rels
from bbl_organizations o
left join lateral (
  select
   	organization_id,
    json_agg(json_build_object('id', o_r.id, 'kind', o_r.kind, 'organization_id', o_r.rel_organization_id) order by o_r.idx) filter (where o_r.id is not null) as rels
  from bbl_organizations_rels o_r
  where o_r.organization_id=o.id
  group by organization_id
) o_r on o_r.organization_id=o.id;

create view bbl_works_view as
select w.*,
       w_c.contributors as contributors,
       w_r.rels as rels
from bbl_works w
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('id', w_c.id, 'person_id', w_c.person_id, 'person', p, 'attrs', w_c.attrs) order by w_c.idx) filter (where w_c.id is not null) as contributors
  from bbl_works_contributors w_c
  left join bbl_people p on p.id = w_c.person_id
  where w_c.work_id=w.id
  group by work_id
) w_c on w_c.work_id=w.id
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('id', w_r.id, 'kind', w_r.kind, 'work_id', w_r.work_id) order by w_r.idx) filter (where w_r.id is not null) as rels
  from bbl_works_rels w_r
  where w_r.work_id=w.id
  group by work_id
) w_r on w_r.work_id=w.id;

-- +goose Down

drop view bbl_organizations_view;
drop view bbl_works_view;