package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// refSubquery builds the subquery part of a ref resolution.
func refSubquery(ref Ref, source, entityTable, sourcesTable, sourceFK, identifiersTable, identifierFK string) (string, []any, error) {
	switch {
	case ref.ID != nil:
		return fmt.Sprintf(`SELECT id FROM %s WHERE id = $1`, entityTable), []any{*ref.ID}, nil
	case ref.SourceID != "":
		return fmt.Sprintf(`SELECT %s FROM %s WHERE source = $1 AND source_id = $2`, sourceFK, sourcesTable), []any{source, ref.SourceID}, nil
	case ref.Identifier != nil:
		return fmt.Sprintf(`SELECT %s FROM %s WHERE scheme = $1 AND val = $2 LIMIT 1`, identifierFK, identifiersTable), []any{ref.Identifier.Scheme, ref.Identifier.Val}, nil
	default:
		return "", nil, fmt.Errorf("empty ref")
	}
}

func resolveWorkRef(ctx context.Context, tx pgx.Tx, ref Ref, source string) (*Work, error) {
	sub, args, err := refSubquery(ref, source, "bbl_works", "bbl_work_sources", "work_id", "bbl_work_identifiers", "work_id")
	if err != nil {
		return nil, fmt.Errorf("resolveWorkRef: %w", err)
	}
	row := tx.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works WHERE id = (`+sub+`)`, args...)
	w, err := scanWork(row)
	if err != nil {
		return nil, fmt.Errorf("resolveWorkRef: %w", err)
	}
	return w, nil
}

func resolveProjectRef(ctx context.Context, tx pgx.Tx, ref Ref, source string) (*Project, error) {
	sub, args, err := refSubquery(ref, source, "bbl_projects", "bbl_project_sources", "project_id", "bbl_project_identifiers", "project_id")
	if err != nil {
		return nil, fmt.Errorf("resolveProjectRef: %w", err)
	}
	row := tx.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_projects WHERE id = (`+sub+`)`, args...)
	p, err := scanProject(row)
	if err != nil {
		return nil, fmt.Errorf("resolveProjectRef: %w", err)
	}
	return p, nil
}

func resolveOrganizationRef(ctx context.Context, tx pgx.Tx, ref Ref, source string) (*Organization, error) {
	sub, args, err := refSubquery(ref, source, "bbl_organizations", "bbl_organization_sources", "organization_id", "bbl_organization_identifiers", "organization_id")
	if err != nil {
		return nil, fmt.Errorf("resolveOrganizationRef: %w", err)
	}
	row := tx.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_organizations WHERE id = (`+sub+`)`, args...)
	o, err := scanOrganization(row)
	if err != nil {
		return nil, fmt.Errorf("resolveOrganizationRef: %w", err)
	}
	return o, nil
}

func resolvePersonRef(ctx context.Context, tx pgx.Tx, ref Ref, source string) (*Person, error) {
	sub, args, err := refSubquery(ref, source, "bbl_people", "bbl_person_sources", "person_id", "bbl_person_identifiers", "person_id")
	if err != nil {
		return nil, fmt.Errorf("resolvePersonRef: %w", err)
	}
	row := tx.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, deleted_at, deleted_by_id,
		       cache
		FROM bbl_people WHERE id = (`+sub+`)`, args...)
	p, err := scanPerson(row)
	if err != nil {
		return nil, fmt.Errorf("resolvePersonRef: %w", err)
	}
	return p, nil
}

// autoPinAllWork runs auto-pin for all assertion tables of a work.
// Used after import to recalculate pinning for all grouping keys.
func autoPinAllWork(ctx context.Context, tx pgx.Tx, workID ID, priorities map[string]int) error {
	// Scalar fields: auto-pin each distinct field.
	fields, err := distinctValues(ctx, tx, "bbl_work_fields", "field", "work_id", workID)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if err := autoPinScalar(ctx, tx, "bbl_work_fields", "work_id", workID, f, "work_source_id", "bbl_work_sources", priorities); err != nil {
			return err
		}
	}

	// All relation tables are collective.
	collectiveTables := []string{
		"bbl_work_identifiers",
		"bbl_work_classifications",
		"bbl_work_titles",
		"bbl_work_abstracts",
		"bbl_work_lay_summaries",
		"bbl_work_contributors",
		"bbl_work_notes",
		"bbl_work_keywords",
		"bbl_work_projects",
		"bbl_work_organizations",
		"bbl_work_rels",
	}
	for _, t := range collectiveTables {
		if err := autoPinCollective(ctx, tx, t, "work_id", workID, "work_source_id", "bbl_work_sources", priorities); err != nil {
			return err
		}
	}

	return nil
}

// autoPinAllPerson runs auto-pin for all assertion tables of a person.
func autoPinAllPerson(ctx context.Context, tx pgx.Tx, personID ID, priorities map[string]int) error {
	// Scalar fields.
	fields, err := distinctValues(ctx, tx, "bbl_person_fields", "field", "person_id", personID)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if err := autoPinScalar(ctx, tx, "bbl_person_fields", "person_id", personID, f, "person_source_id", "bbl_person_sources", priorities); err != nil {
			return err
		}
	}

	// Collective tables.
	collectiveTables := []string{
		"bbl_person_identifiers",
		"bbl_person_organizations",
	}
	for _, t := range collectiveTables {
		if err := autoPinCollective(ctx, tx, t, "person_id", personID, "person_source_id", "bbl_person_sources", priorities); err != nil {
			return err
		}
	}

	return nil
}

// autoPinAllProject runs auto-pin for all assertion tables of a project.
func autoPinAllProject(ctx context.Context, tx pgx.Tx, projectID ID, priorities map[string]int) error {
	// Scalar fields.
	fields, err := distinctValues(ctx, tx, "bbl_project_fields", "field", "project_id", projectID)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if err := autoPinScalar(ctx, tx, "bbl_project_fields", "project_id", projectID, f, "project_source_id", "bbl_project_sources", priorities); err != nil {
			return err
		}
	}

	// Collective tables.
	collectiveTables := []string{
		"bbl_project_titles",
		"bbl_project_descriptions",
		"bbl_project_identifiers",
		"bbl_project_people",
	}
	for _, t := range collectiveTables {
		if err := autoPinCollective(ctx, tx, t, "project_id", projectID, "project_source_id", "bbl_project_sources", priorities); err != nil {
			return err
		}
	}

	return nil
}

// autoPinAllOrganization runs auto-pin for all assertion tables of an organization.
func autoPinAllOrganization(ctx context.Context, tx pgx.Tx, orgID ID, priorities map[string]int) error {
	// Organization has no str_fields (kind is a column, not a field assertion).
	// All tables are collective.
	collectiveTables := []string{
		"bbl_organization_identifiers",
		"bbl_organization_names",
		"bbl_organization_rels",
	}
	for _, t := range collectiveTables {
		if err := autoPinCollective(ctx, tx, t, "organization_id", orgID, "organization_source_id", "bbl_organization_sources", priorities); err != nil {
			return err
		}
	}

	return nil
}

// distinctValues returns the distinct values of a column for an entity.
func distinctValues(ctx context.Context, tx pgx.Tx, table, col, entityIDCol string, entityID ID) ([]string, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT DISTINCT %s FROM %s WHERE %s = $1`, col, table, entityIDCol), entityID)
	if err != nil {
		return nil, fmt.Errorf("distinctValues: %w", err)
	}
	defer rows.Close()
	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("distinctValues: %w", err)
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// workAssertionTables lists all assertion tables for works (used for bulk delete on re-import).
var workAssertionTables = []string{
	"bbl_work_fields",
	"bbl_work_identifiers",
	"bbl_work_classifications",
	"bbl_work_contributors",
	"bbl_work_titles",
	"bbl_work_abstracts",
	"bbl_work_lay_summaries",
	"bbl_work_notes",
	"bbl_work_keywords",
	"bbl_work_projects",
	"bbl_work_organizations",
	"bbl_work_rels",
}

// personAssertionTables lists all assertion tables for people.
var personAssertionTables = []string{
	"bbl_person_fields",
	"bbl_person_identifiers",
	"bbl_person_organizations",
}

// projectAssertionTables lists all assertion tables for projects.
var projectAssertionTables = []string{
	"bbl_project_fields",
	"bbl_project_titles",
	"bbl_project_descriptions",
	"bbl_project_identifiers",
	"bbl_project_people",
}

// organizationAssertionTables lists all assertion tables for organizations.
var organizationAssertionTables = []string{
	"bbl_organization_fields",
	"bbl_organization_identifiers",
	"bbl_organization_names",
	"bbl_organization_rels",
}

// deleteSourceAssertions deletes all assertions linked to a source record.
// sourceRecordID is the uuid PK of the *_sources row. sourceIDCol is the FK
// column in assertion tables (e.g. "work_source_id").
func deleteSourceAssertions(ctx context.Context, tx pgx.Tx, tables []string, sourceIDCol string, sourceRecordID ID) error {
	for _, table := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`DELETE FROM %s WHERE %s = $1`, table, sourceIDCol),
			sourceRecordID); err != nil {
			return fmt.Errorf("deleteSourceAssertions(%s): %w", table, err)
		}
	}
	return nil
}

// rebuildPersonCache rebuilds the cache column for the given person IDs from pinned assertions.
func rebuildPersonCache(ctx context.Context, tx pgx.Tx, personIDs []ID) error {
	if len(personIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE bbl_people p
		SET cache = json_build_object(
			'name',        COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_fields sf WHERE sf.person_id = p.id AND sf.field = 'name' AND sf.pinned = true), ''),
			'given_name',  COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_fields sf WHERE sf.person_id = p.id AND sf.field = 'given_name' AND sf.pinned = true), ''),
			'middle_name', COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_fields sf WHERE sf.person_id = p.id AND sf.field = 'middle_name' AND sf.pinned = true), ''),
			'family_name', COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_fields sf WHERE sf.person_id = p.id AND sf.field = 'family_name' AND sf.pinned = true), ''),
			'identifiers', (SELECT json_agg(json_build_object('scheme', i.scheme, 'val', i.val) ORDER BY i.scheme, i.val) FROM bbl_person_identifiers i WHERE i.person_id = p.id AND i.pinned = true),
			'organizations', (SELECT json_agg(json_build_object('organization_id', po.organization_id) ORDER BY po.organization_id) FROM bbl_person_organizations po WHERE po.person_id = p.id AND po.pinned = true)
		)
		WHERE p.id = ANY($1)`, dedup(personIDs))
	if err != nil {
		return fmt.Errorf("rebuildPersonCache: %w", err)
	}
	return nil
}

// rebuildProjectCache rebuilds the cache column for the given project IDs from pinned assertions.
func rebuildProjectCache(ctx context.Context, tx pgx.Tx, projectIDs []ID) error {
	if len(projectIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE bbl_projects p
		SET cache = json_build_object(
			'titles', (SELECT json_agg(json_build_object('lang', t.lang, 'val', t.val) ORDER BY t.lang, t.val) FROM bbl_project_titles t WHERE t.project_id = p.id AND t.pinned = true),
			'descriptions', (SELECT json_agg(json_build_object('lang', d.lang, 'val', d.val) ORDER BY d.lang, d.val) FROM bbl_project_descriptions d WHERE d.project_id = p.id AND d.pinned = true),
			'identifiers', (SELECT json_agg(json_build_object('scheme', i.scheme, 'val', i.val) ORDER BY i.scheme, i.val) FROM bbl_project_identifiers i WHERE i.project_id = p.id AND i.pinned = true),
			'people', (SELECT json_agg(json_build_object('person_id', pp.person_id, 'role', pp.role) ORDER BY pp.person_id) FROM bbl_project_people pp WHERE pp.project_id = p.id AND pp.pinned = true)
		)
		WHERE p.id = ANY($1)`, dedup(projectIDs))
	if err != nil {
		return fmt.Errorf("rebuildProjectCache: %w", err)
	}
	return nil
}

// rebuildOrganizationCache rebuilds the cache column for the given organization IDs from pinned assertions.
func rebuildOrganizationCache(ctx context.Context, tx pgx.Tx, orgIDs []ID) error {
	if len(orgIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE bbl_organizations o
		SET cache = json_build_object(
			'identifiers', (SELECT json_agg(json_build_object('scheme', i.scheme, 'val', i.val) ORDER BY i.scheme, i.val) FROM bbl_organization_identifiers i WHERE i.organization_id = o.id AND i.pinned = true),
			'names', (SELECT json_agg(json_build_object('lang', t.lang, 'val', t.val) ORDER BY t.lang, t.val) FROM bbl_organization_names t WHERE t.organization_id = o.id AND t.pinned = true),
			'rels', (SELECT json_agg(json_build_object('id', r.id, 'organization_id', r.organization_id, 'rel_organization_id', r.rel_organization_id, 'kind', r.kind, 'start_date', r.start_date, 'end_date', r.end_date) ORDER BY r.kind, r.rel_organization_id) FROM bbl_organization_rels r WHERE r.organization_id = o.id AND r.pinned = true)
		)
		WHERE o.id = ANY($1)`, dedup(orgIDs))
	if err != nil {
		return fmt.Errorf("rebuildOrganizationCache: %w", err)
	}
	return nil
}

// rebuildWorkCache rebuilds the cache column for the given work IDs from the view.
func rebuildWorkCache(ctx context.Context, tx pgx.Tx, workIDs []ID) error {
	if len(workIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE bbl_works w
		SET cache = json_build_object(
			'str_fields', v.str_fields,
			'identifiers', v.identifiers,
			'classifications', v.classifications,
			'contributors', v.contributors,
			'titles', v.titles,
			'abstracts', v.abstracts,
			'lay_summaries', v.lay_summaries,
			'notes', v.notes,
			'keywords', v.keywords
		)
		FROM bbl_works_view v
		WHERE v.id = w.id AND w.id = ANY($1)`, dedup(workIDs))
	if err != nil {
		return fmt.Errorf("rebuildWorkCache: %w", err)
	}
	return nil
}
