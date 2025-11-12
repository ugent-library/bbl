-- +goose up

CREATE VIEW bbl_users_view AS
SELECT u.*,
       u_i.identifiers AS identifiers
FROM bbl_users u
LEFT JOIN LATERAL (
  SELECT
   	user_id,
    json_agg(json_build_object('scheme', u_i.scheme, 'val', u_i.val) ORDER BY u_i.idx) FILTER (WHERE u_i.idx IS NOT NULL) AS identifiers
  FROM bbl_user_identifiers u_i
  WHERE u_i.user_id = u.id
  GROUP BY user_id
) u_i ON u_i.user_id = u.id;

CREATE VIEW bbl_organizations_view AS
SELECT o.*,
       row_to_json(u_c) AS created_by,
       row_to_json(u_u) AS updated_by,
       o_i.identifiers AS identifiers,
       o_r.rels AS rels
FROM bbl_organizations o
LEFT JOIN bbl_users_view u_c ON o.created_by_id = u_c.id
LEFT JOIN bbl_users_view u_u ON o.updated_by_id = u_u.id
LEFT JOIN LATERAL (
  SELECT
   	organization_id,
    json_agg(json_build_object('scheme', o_i.scheme, 'val', o_i.val) ORDER BY o_i.idx) FILTER (WHERE o_i.idx IS NOT NULL) AS identifiers
  FROM bbl_organization_identifiers o_i
  WHERE o_i.organization_id = o.id
  GROUP BY organization_id
) o_i ON o_i.organization_id = o.id
LEFT JOIN LATERAL (
  SELECT
   	organization_id,
    json_agg(json_build_object('kind', o_r.kind, 'organization_id', o_r.rel_organization_id) ORDER BY o_r.idx) FILTER (WHERE o_r.idx IS NOT NULL) AS rels
  FROM bbl_organization_rels o_r
  WHERE o_r.organization_id = o.id
  GROUP BY organization_id
) o_r ON o_r.organization_id = o.id;

CREATE VIEW bbl_people_view AS
SELECT p.*,
       row_to_json(u_c) AS created_by,
       row_to_json(u_u) AS updated_by,
       p_i.identifiers AS identifiers
FROM bbl_people p
LEFT JOIN bbl_users_view u_c ON p.created_by_id = u_c.id
LEFT JOIN bbl_users_view u_u ON p.updated_by_id = u_u.id
LEFT JOIN LATERAL (
  SELECT
   	person_id,
    json_agg(json_build_object('scheme', p_i.scheme, 'val', p_i.val) ORDER BY p_i.idx) FILTER (WHERE p_i.idx IS NOT NULL) AS identifiers
  FROM bbl_person_identifiers p_i
  WHERE p_i.person_id = p.id
  GROUP BY person_id
) p_i ON p_i.person_id = p.id;

CREATE VIEW bbl_projects_view AS
SELECT p.*,
       row_to_json(u_c) AS created_by,
       row_to_json(u_u) AS updated_by,
       p_i.identifiers AS identifiers
FROM bbl_projects p
LEFT JOIN bbl_users_view u_c ON p.created_by_id = u_c.id
LEFT JOIN bbl_users_view u_u ON p.updated_by_id = u_u.id
LEFT JOIN LATERAL (
  SELECT
   	project_id,
    json_agg(json_build_object('scheme', p_i.scheme, 'val', p_i.val) ORDER BY p_i.idx) FILTER (WHERE p_i.idx IS NOT NULL) AS identifiers
  FROM bbl_project_identifiers p_i
  WHERE p_i.project_id = p.id
  GROUP BY project_id
) p_i ON p_i.project_id = p.id;

CREATE VIEW bbl_works_view AS
SELECT w.*,
       row_to_json(u_c) AS created_by,
       row_to_json(u_u) AS updated_by,
       w_pe.permissions AS permissions,
       w_i.identifiers AS identifiers,
       w_c.contributors AS contributors,
       w_f.files AS files,
       w_r.rels AS rels
FROM bbl_works w
LEFT JOIN bbl_users_view u_c ON w.created_by_id = u_c.id
LEFT JOIN bbl_users_view u_u ON w.updated_by_id = u_u.id
LEFT JOIN LATERAL (
  SELECT
   	work_id,
    json_agg(json_build_object('kind', w_pe.kind, 'user_id', w_pe.user_id)) FILTER (WHERE w_pe.work_id IS NOT NULL) AS permissions
  FROM bbl_work_permissions w_pe
  WHERE w_pe.work_id = w.id
  GROUP BY work_id
) w_pe ON w_pe.work_id = w.id
LEFT JOIN LATERAL (
  SELECT
   	work_id,
    json_agg(json_build_object('scheme', w_i.scheme, 'val', w_i.val) ORDER BY w_i.idx) FILTER (WHERE w_i.idx IS NOT NULL) AS identifiers
  FROM bbl_work_identifiers w_i
  WHERE w_i.work_id=w.id
  GROUP BY work_id
) w_i ON w_i.work_id=w.id
LEFT JOIN LATERAL (
  SELECT
   	work_id,
    json_agg(json_build_object('person_id', w_c.person_id, 'person', p, 'attrs', w_c.attrs) ORDER BY w_c.idx) FILTER (WHERE w_c.idx IS NOT NULL) AS contributors
  FROM bbl_work_contributors w_c
  LEFT JOIN bbl_people_view p ON p.id = w_c.person_id
  WHERE w_c.work_id = w.id
  GROUP BY work_id
) w_c ON w_c.work_id = w.id
LEFT JOIN LATERAL (
  SELECT
   	work_id,
    json_agg(json_build_object('object_id', w_f.object_id, 'name', w_f.name, 'content_type', w_f.content_type, 'size', w_f.size) ORDER BY w_f.idx) FILTER (WHERE w_f.idx IS NOT NULL) AS files
  FROM bbl_work_files w_f
  WHERE w_f.work_id = w.id
  GROUP BY work_id
) w_f ON w_f.work_id = w.id
LEFT JOIN LATERAL (
  SELECT
   	work_id,
    json_agg(json_build_object('kind', w_r.kind, 'work_id', w_r.rel_work_id) ORDER BY w_r.idx) FILTER (WHERE w_r.idx IS NOT NULL) AS rels
  FROM bbl_work_rels w_r
  WHERE w_r.work_id = w.id
  GROUP BY work_id
) w_r ON w_r.work_id = w.id;

CREATE VIEW bbl_representations_view AS
SELECT r.*,
       s_r.sets AS sets
FROM bbl_representations r
LEFT JOIN LATERAL (
  SELECT
   	representation_id,
    array_agg(s.name) FILTER (WHERE s.id IS NOT NULL) AS sets
  FROM bbl_set_representations s_r
  LEFT JOIN bbl_sets s ON s.id = s_r.set_id
  WHERE s_r.representation_id = r.id
  GROUP BY representation_id
) s_r ON s_r.representation_id = r.id;

-- +goose down

DROP VIEW bbl_representations_view;
DROP VIEW bbl_works_view;
DROP VIEW bbl_people_view;
DROP VIEW bbl_organizations_view;
DROP VIEW bbl_projects_view;
DROP VIEW bbl_users_view;
