-- +goose up

CREATE TABLE bbl_organization_searches (
  query text PRIMARY KEY,
  total bigint NOT NULL
);

CREATE TABLE bbl_person_searches (
  query text PRIMARY KEY,
  total bigint NOT NULL
);

CREATE TABLE bbl_project_searches (
  query text PRIMARY KEY,
  total bigint NOT NULL
);

CREATE TABLE bbl_work_searches (
  query text PRIMARY KEY,
  total bigint NOT NULL
);

-- +goose down

DROP TABLE bbl_organization_searches;
DROP TABLE bbl_person_searches;
DROP TABLE bbl_project_searches;
DROP TABLE bbl_work_searches;
