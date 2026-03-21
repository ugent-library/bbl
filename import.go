package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// refSubquery builds the subquery part of a ref resolution.
func refSubquery(ref Ref, source, entityTable, sourcesTable, sourceFK, assertionsTable, entityIDCol string) (string, []any, error) {
	switch {
	case ref.ID != nil:
		return fmt.Sprintf(`SELECT id FROM %s WHERE id = $1`, entityTable), []any{*ref.ID}, nil
	case ref.SourceID != "":
		return fmt.Sprintf(`SELECT %s FROM %s WHERE source = $1 AND source_id = $2`, sourceFK, sourcesTable), []any{source, ref.SourceID}, nil
	case ref.Identifier != nil:
		return fmt.Sprintf(
			`SELECT %s FROM %s WHERE field = 'identifiers' AND val->>'scheme' = $1 AND val->>'val' = $2 LIMIT 1`,
			entityIDCol, assertionsTable), []any{ref.Identifier.Scheme, ref.Identifier.Val}, nil
	default:
		return "", nil, fmt.Errorf("empty ref")
	}
}

func resolveWorkRef(ctx context.Context, tx pgx.Tx, ref Ref, source string) (*Work, error) {
	sub, args, err := refSubquery(ref, source, "bbl_works", "bbl_work_sources", "work_id", "bbl_work_assertions", "work_id")
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
	sub, args, err := refSubquery(ref, source, "bbl_projects", "bbl_project_sources", "project_id", "bbl_project_assertions", "project_id")
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
	sub, args, err := refSubquery(ref, source, "bbl_organizations", "bbl_organization_sources", "organization_id", "bbl_organization_assertions", "organization_id")
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
	sub, args, err := refSubquery(ref, source, "bbl_people", "bbl_person_sources", "person_id", "bbl_person_assertions", "person_id")
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

// logHistory logs the old value of a human assertion before it is replaced or deleted.
// Called by Set/Hide/Unset updaters before DELETE.
func logHistory(ctx context.Context, tx pgx.Tx, recordType string, assertionsTable, entityIDCol string, entityID ID, field string, revID int64) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO bbl_history (rev_id, record_type, record_id, field, val, hidden)
		SELECT $1, $2, a.%s, a.field, a.val, a.hidden
		FROM %s a
		WHERE a.%s = $3 AND a.field = $4 AND a.user_id IS NOT NULL`,
		entityIDCol, assertionsTable, entityIDCol),
		revID, recordType, entityID, field)
	if err != nil {
		return fmt.Errorf("logHistory(%s.%s): %w", recordType, field, err)
	}
	return nil
}

// Convenience wrappers per entity type.
func logWorkHistory(ctx context.Context, tx pgx.Tx, workID ID, field string, revID int64) error {
	return logHistory(ctx, tx, RecordTypeWork, "bbl_work_assertions", "work_id", workID, field, revID)
}

func logPersonHistory(ctx context.Context, tx pgx.Tx, personID ID, field string, revID int64) error {
	return logHistory(ctx, tx, RecordTypePerson, "bbl_person_assertions", "person_id", personID, field, revID)
}

func logProjectHistory(ctx context.Context, tx pgx.Tx, projectID ID, field string, revID int64) error {
	return logHistory(ctx, tx, RecordTypeProject, "bbl_project_assertions", "project_id", projectID, field, revID)
}

func logOrganizationHistory(ctx context.Context, tx pgx.Tx, orgID ID, field string, revID int64) error {
	return logHistory(ctx, tx, RecordTypeOrganization, "bbl_organization_assertions", "organization_id", orgID, field, revID)
}

// parseAssertionsInfo extracts the assertions_info key from a cache JSON blob.
func parseAssertionsInfo(cache []byte) map[string][]assertionInfo {
	if len(cache) == 0 {
		return nil
	}
	var d struct {
		AssertionsInfo map[string][]assertionInfo `json:"assertions_info"`
	}
	json.Unmarshal(cache, &d)
	return d.AssertionsInfo
}

// autoPinAll runs auto-pin for all fields of an entity.
// Gets distinct fields from the assertions table and calls autoPin for each.
func autoPinAll(ctx context.Context, tx pgx.Tx, assertionsTable, entityIDCol string, entityID ID, sourceIDCol, sourceTable string, priorities map[string]int) error {
	fields, err := distinctValues(ctx, tx, assertionsTable, "field", entityIDCol, entityID)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if err := autoPin(ctx, tx, assertionsTable, entityIDCol, entityID, f, sourceIDCol, sourceTable, priorities); err != nil {
			return err
		}
	}
	return nil
}

func autoPinAllWork(ctx context.Context, tx pgx.Tx, workID ID, priorities map[string]int) error {
	return autoPinAll(ctx, tx, "bbl_work_assertions", "work_id", workID, "work_source_id", "bbl_work_sources", priorities)
}

func autoPinAllPerson(ctx context.Context, tx pgx.Tx, personID ID, priorities map[string]int) error {
	return autoPinAll(ctx, tx, "bbl_person_assertions", "person_id", personID, "person_source_id", "bbl_person_sources", priorities)
}

func autoPinAllProject(ctx context.Context, tx pgx.Tx, projectID ID, priorities map[string]int) error {
	return autoPinAll(ctx, tx, "bbl_project_assertions", "project_id", projectID, "project_source_id", "bbl_project_sources", priorities)
}

func autoPinAllOrganization(ctx context.Context, tx pgx.Tx, orgID ID, priorities map[string]int) error {
	return autoPinAll(ctx, tx, "bbl_organization_assertions", "organization_id", orgID, "organization_source_id", "bbl_organization_sources", priorities)
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

// deleteSourceAssertions deletes all assertions linked to a source record.
// CASCADE on the assertions table handles relation table cleanup.
func deleteSourceAssertions(ctx context.Context, tx pgx.Tx, assertionsTable, sourceIDCol string, sourceRecordID ID) error {
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE %s = $1`, assertionsTable, sourceIDCol),
		sourceRecordID); err != nil {
		return fmt.Errorf("deleteSourceAssertions(%s): %w", assertionsTable, err)
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
			'name',        COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_assertions sf WHERE sf.person_id = p.id AND sf.field = 'name' AND sf.pinned = true AND NOT sf.hidden), ''),
			'given_name',  COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_assertions sf WHERE sf.person_id = p.id AND sf.field = 'given_name' AND sf.pinned = true AND NOT sf.hidden), ''),
			'middle_name', COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_assertions sf WHERE sf.person_id = p.id AND sf.field = 'middle_name' AND sf.pinned = true AND NOT sf.hidden), ''),
			'family_name', COALESCE((SELECT sf.val #>> '{}' FROM bbl_person_assertions sf WHERE sf.person_id = p.id AND sf.field = 'family_name' AND sf.pinned = true AND NOT sf.hidden), ''),
			'identifiers', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_person_assertions a WHERE a.person_id = p.id AND a.field = 'identifiers' AND a.pinned = true AND NOT a.hidden),
			'affiliations', (SELECT json_agg(json_build_object('val', a.val, 'organization_id', po.organization_id) ORDER BY a.id) FROM bbl_person_assertions a LEFT JOIN bbl_person_assertion_affiliations po ON po.assertion_id = a.id WHERE a.person_id = p.id AND a.field = 'affiliations' AND a.pinned = true AND NOT a.hidden),
			'assertions_info', (SELECT json_object_agg(sub.field, sub.infos) FROM (SELECT a.field, json_agg(json_build_object('human', a.user_id IS NOT NULL, 'role', a.role, 'hidden', a.hidden, 'pinned', a.pinned, 'source', s.source) ORDER BY a.id) AS infos FROM bbl_person_assertions a LEFT JOIN bbl_person_sources s ON s.id = a.person_source_id WHERE a.person_id = p.id AND a.pinned = true GROUP BY a.field) sub)
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
			'titles', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_project_assertions a WHERE a.project_id = p.id AND a.field = 'titles' AND a.pinned = true AND NOT a.hidden),
			'descriptions', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_project_assertions a WHERE a.project_id = p.id AND a.field = 'descriptions' AND a.pinned = true AND NOT a.hidden),
			'identifiers', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_project_assertions a WHERE a.project_id = p.id AND a.field = 'identifiers' AND a.pinned = true AND NOT a.hidden),
			'participants', (SELECT json_agg(json_build_object('val', a.val, 'person_id', pp.person_id, 'role', pp.role) ORDER BY a.id) FROM bbl_project_assertions a LEFT JOIN bbl_project_assertion_participants pp ON pp.assertion_id = a.id WHERE a.project_id = p.id AND a.field = 'participants' AND a.pinned = true AND NOT a.hidden),
			'assertions_info', (SELECT json_object_agg(sub.field, sub.infos) FROM (SELECT a.field, json_agg(json_build_object('human', a.user_id IS NOT NULL, 'role', a.role, 'hidden', a.hidden, 'pinned', a.pinned, 'source', s.source) ORDER BY a.id) AS infos FROM bbl_project_assertions a LEFT JOIN bbl_project_sources s ON s.id = a.project_source_id WHERE a.project_id = p.id AND a.pinned = true GROUP BY a.field) sub)
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
			'identifiers', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_organization_assertions a WHERE a.organization_id = o.id AND a.field = 'identifiers' AND a.pinned = true AND NOT a.hidden),
			'names', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_organization_assertions a WHERE a.organization_id = o.id AND a.field = 'names' AND a.pinned = true AND NOT a.hidden),
			'rels', (SELECT json_agg(json_build_object('val', a.val, 'rel_organization_id', r.rel_organization_id, 'kind', r.kind, 'start_date', r.start_date, 'end_date', r.end_date) ORDER BY r.kind, r.rel_organization_id) FROM bbl_organization_assertions a LEFT JOIN bbl_organization_assertion_rels r ON r.assertion_id = a.id WHERE a.organization_id = o.id AND a.field = 'rels' AND a.pinned = true AND NOT a.hidden),
			'assertions_info', (SELECT json_object_agg(sub.field, sub.infos) FROM (SELECT a.field, json_agg(json_build_object('human', a.user_id IS NOT NULL, 'role', a.role, 'hidden', a.hidden, 'pinned', a.pinned, 'source', s.source) ORDER BY a.id) AS infos FROM bbl_organization_assertions a LEFT JOIN bbl_organization_sources s ON s.id = a.organization_source_id WHERE a.organization_id = o.id AND a.pinned = true GROUP BY a.field) sub)
		)
		WHERE o.id = ANY($1)`, dedup(orgIDs))
	if err != nil {
		return fmt.Errorf("rebuildOrganizationCache: %w", err)
	}
	return nil
}

// rebuildWorkCache rebuilds the cache column for the given work IDs from pinned assertions.
func rebuildWorkCache(ctx context.Context, tx pgx.Tx, workIDs []ID) error {
	if len(workIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE bbl_works w
		SET cache = json_build_object(
			'str_fields', (SELECT json_agg(json_build_object('field', a.field, 'val', a.val) ORDER BY a.field) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.pinned = true AND NOT a.hidden AND a.field NOT IN ('identifiers', 'classifications', 'titles', 'abstracts', 'lay_summaries', 'notes', 'keywords', 'contributors', 'projects', 'organizations', 'rels')),
			'identifiers', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'identifiers' AND a.pinned = true AND NOT a.hidden),
			'classifications', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'classifications' AND a.pinned = true AND NOT a.hidden),
			'titles', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'titles' AND a.pinned = true AND NOT a.hidden),
			'abstracts', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'abstracts' AND a.pinned = true AND NOT a.hidden),
			'lay_summaries', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'lay_summaries' AND a.pinned = true AND NOT a.hidden),
			'notes', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'notes' AND a.pinned = true AND NOT a.hidden),
			'keywords', (SELECT json_agg(a.val ORDER BY a.id) FROM bbl_work_assertions a WHERE a.work_id = w.id AND a.field = 'keywords' AND a.pinned = true AND NOT a.hidden),
			'contributors', (SELECT json_agg(json_build_object('val', a.val, 'person_id', c.person_id, 'organization_id', c.organization_id) ORDER BY a.id) FROM bbl_work_assertions a LEFT JOIN bbl_work_assertion_contributors c ON c.assertion_id = a.id WHERE a.work_id = w.id AND a.field = 'contributors' AND a.pinned = true AND NOT a.hidden),
			'projects', (SELECT json_agg(p.project_id ORDER BY a.id) FROM bbl_work_assertions a JOIN bbl_work_assertion_projects p ON p.assertion_id = a.id WHERE a.work_id = w.id AND a.field = 'projects' AND a.pinned = true AND NOT a.hidden),
			'organizations', (SELECT json_agg(o.organization_id ORDER BY a.id) FROM bbl_work_assertions a JOIN bbl_work_assertion_organizations o ON o.assertion_id = a.id WHERE a.work_id = w.id AND a.field = 'organizations' AND a.pinned = true AND NOT a.hidden),
			'rels', (SELECT json_agg(json_build_object('related_work_id', r.related_work_id, 'kind', r.kind) ORDER BY a.id) FROM bbl_work_assertions a JOIN bbl_work_assertion_rels r ON r.assertion_id = a.id WHERE a.work_id = w.id AND a.field = 'rels' AND a.pinned = true AND NOT a.hidden),
			'assertions_info', (SELECT json_object_agg(sub.field, sub.infos) FROM (SELECT a.field, json_agg(json_build_object('human', a.user_id IS NOT NULL, 'role', a.role, 'hidden', a.hidden, 'pinned', a.pinned, 'source', s.source) ORDER BY a.id) AS infos FROM bbl_work_assertions a LEFT JOIN bbl_work_sources s ON s.id = a.work_source_id WHERE a.work_id = w.id AND a.pinned = true GROUP BY a.field) sub)
		)
		WHERE w.id = ANY($1)`, dedup(workIDs))
	if err != nil {
		return fmt.Errorf("rebuildWorkCache: %w", err)
	}
	return nil
}
