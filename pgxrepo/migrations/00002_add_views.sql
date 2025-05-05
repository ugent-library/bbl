-- +goose up

create view bbl_organizations_view as
select o.*,
       o_i.identifiers as identifiers,
       o_r.rels as rels
from bbl_organizations o
left join lateral (
  select
   	organization_id,
    json_agg(json_build_object('scheme', o_i.scheme, 'val', o_i.val) order by o_i.idx) filter (where o_i.idx is not null) as identifiers
  from bbl_organizations_identifiers o_i
  where o_i.organization_id=o.id
  group by organization_id
) o_i on o_i.organization_id=o.id
left join lateral (
  select
   	organization_id,
    json_agg(json_build_object('kind', o_r.kind, 'organization_id', o_r.rel_organization_id) order by o_r.idx) filter (where o_r.idx is not null) as rels
  from bbl_organizations_rels o_r
  where o_r.organization_id=o.id
  group by organization_id
) o_r on o_r.organization_id=o.id;

create view bbl_people_view as
select p.*,
       p_i.identifiers as identifiers
from bbl_people p
left join lateral (
  select
   	person_id,
    json_agg(json_build_object('scheme', p_i.scheme, 'val', p_i.val) order by p_i.idx) filter (where p_i.idx is not null) as identifiers
  from bbl_people_identifiers p_i
  where p_i.person_id=p.id
  group by person_id
) p_i on p_i.person_id=p.id;

create view bbl_projects_view as
select p.*,
       p_i.identifiers as identifiers
from bbl_projects p
left join lateral (
  select
   	project_id,
    json_agg(json_build_object('scheme', p_i.scheme, 'val', p_i.val) order by p_i.idx) filter (where p_i.idx is not null) as identifiers
  from bbl_projects_identifiers p_i
  where p_i.project_id=p.id
  group by project_id
) p_i on p_i.project_id=p.id;

create view bbl_works_view as
select w.*,
       w_i.identifiers as identifiers,
       w_c.contributors as contributors,
       w_f.files as files,
       w_r.rels as rels
from bbl_works w
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('scheme', w_i.scheme, 'val', w_i.val) order by w_i.idx) filter (where w_i.idx is not null) as identifiers
  from bbl_works_identifiers w_i
  where w_i.work_id=w.id
  group by work_id
) w_i on w_i.work_id=w.id
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('person_id', w_c.person_id, 'person', p, 'attrs', w_c.attrs) order by w_c.idx) filter (where w_c.idx is not null) as contributors
  from bbl_works_contributors w_c
  left join bbl_people_view p on p.id = w_c.person_id
  where w_c.work_id=w.id
  group by work_id
) w_c on w_c.work_id=w.id
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('id', w_f.id, 'name', w_f.name, 'content_type', w_f.content_type, 'size', w_f.size) order by w_f.idx) filter (where w_f.idx is not null) as files
  from bbl_work_files w_f
  where w_f.work_id=w.id
  group by work_id
) w_f on w_f.work_id=w.id
left join lateral (
  select
   	work_id,
    json_agg(json_build_object('kind', w_r.kind, 'work_id', w_r.rel_work_id) order by w_r.idx) filter (where w_r.idx is not null) as rels
  from bbl_works_rels w_r
  where w_r.work_id=w.id
  group by work_id
) w_r on w_r.work_id=w.id;

create view bbl_users_view as
select u.*,
       u_i.identifiers as identifiers
from bbl_users u
left join lateral (
  select
   	user_id,
    json_agg(json_build_object('scheme', u_i.scheme, 'val', u_i.val) order by u_i.idx) filter (where u_i.idx is not null) as identifiers
  from bbl_user_identifiers u_i
  where u_i.user_id=u.id
  group by user_id
) u_i on u_i.user_id=u.id;

-- +goose down

drop view bbl_works_view;
drop view bbl_people_view;
drop view bbl_organizations_view;
drop view bbl_projects_view;
drop view bbl_users_view;
