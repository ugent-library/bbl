package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Repo) GetProject(ctx context.Context, id ID) (*Project, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_projects
		WHERE id = $1`, id)
	p, err := scanProject(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetProject: %w", err)
	}
	return p, nil
}

func (r *Repo) ImportProjects(ctx context.Context, source string, seq iter.Seq2[*ImportProjectInput, error]) (int, error) {
	const batchSize = 250
	var pending []*ImportProjectInput
	var total int

	flush := func() error {
		n, err := r.importProjectBatch(ctx, source, pending)
		total += n
		pending = pending[:0]
		return err
	}

	for in, err := range seq {
		if err != nil {
			return total, fmt.Errorf("ImportProjects: %w", err)
		}
		pending = append(pending, in)
		if len(pending) == batchSize {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if len(pending) > 0 {
		if err := flush(); err != nil {
			return total, err
		}
	}
	return total, nil
}

func (r *Repo) importProjectBatch(ctx context.Context, source string, records []*ImportProjectInput) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("importProjectBatch: %w", err)
	}
	defer tx.Rollback(ctx)

	priorities, err := fetchSourcePriorities(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("importProjectBatch: %w", err)
	}

	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (source) VALUES ($1) RETURNING id`,
		source).Scan(&revID); err != nil {
		return 0, fmt.Errorf("importProjectBatch: %w", err)
	}

	var changedProjectIDs []ID
	var n int
	for _, in := range records {
		projectID, isNew, err := r.importProjectRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("importProjectBatch: source_id=%s: %w", in.SourceID, err)
		}
		changedProjectIDs = append(changedProjectIDs, projectID)
		_ = isNew
		n++
	}

	if err := rebuildProjectCache(ctx, tx, changedProjectIDs); err != nil {
		return n, fmt.Errorf("importProjectBatch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("importProjectBatch: %w", err)
	}
	return n, nil
}

func (r *Repo) importProjectRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportProjectInput, revID int64, priorities map[string]int) (ID, bool, error) {
	var projectID ID
	var sourceRecordID ID
	var isNew bool
	err := tx.QueryRow(ctx, `
		SELECT project_id, id FROM bbl_project_sources
		WHERE source = $1 AND source_id = $2
		FOR UPDATE`, source, in.SourceID).Scan(&projectID, &sourceRecordID)
	if errors.Is(err, pgx.ErrNoRows) {
		isNew = true
		if in.ID != nil {
			projectID = *in.ID
		} else {
			projectID = newID()
		}
	} else if err != nil {
		return ID{}, false, err
	}

	if isNew {
		status := in.Status
		if status == "" {
			status = ProjectStatusPublic
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_projects (id, version, status, start_date, end_date)
			VALUES ($1, 1, $2, $3, $4)`,
			projectID, status, in.StartDate, in.EndDate); err != nil {
			return ID{}, false, fmt.Errorf("insert bbl_projects: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_project_sources (id, project_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, projectID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, false, fmt.Errorf("insert bbl_project_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, "bbl_project_assertions", "project_source_id", sourceRecordID); err != nil {
			return ID{}, false, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_project_sources SET record = $1, ingested_at = transaction_timestamp() WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, false, fmt.Errorf("update bbl_project_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_projects SET version = version + 1, updated_at = transaction_timestamp(),
			       start_date = $2, end_date = $3
			WHERE id = $1`, projectID, in.StartDate, in.EndDate); err != nil {
			return ID{}, false, fmt.Errorf("bump version: %w", err)
		}
	}

	// Insert text field assertions.
	if err := importProjectTextFields(ctx, tx, revID, projectID, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Insert relation assertions.
	if err := importProjectRelations(ctx, tx, revID, projectID, source, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Auto-pin all grouping keys.
	if err := autoPinAllProject(ctx, tx, projectID, priorities); err != nil {
		return ID{}, false, err
	}

	return projectID, isNew, nil
}

func importProjectTextFields(ctx context.Context, tx pgx.Tx, revID int64, projectID ID, sourceRecordID ID, in *ImportProjectInput) error {
	if len(in.Titles) > 0 {
		aID, err := writeProjectAssertion(ctx, tx, revID, projectID, "titles", nil, false, &sourceRecordID, nil, nil)
		if err != nil {
			return err
		}
		for _, t := range in.Titles {
			if err := writeProjectTitle(ctx, tx, newID(), projectID, aID, t.Lang, t.Val); err != nil {
				return err
			}
		}
	}
	if len(in.Descriptions) > 0 {
		aID, err := writeProjectAssertion(ctx, tx, revID, projectID, "descriptions", nil, false, &sourceRecordID, nil, nil)
		if err != nil {
			return err
		}
		for _, d := range in.Descriptions {
			if err := writeProjectDescription(ctx, tx, newID(), projectID, aID, d.Lang, d.Val); err != nil {
				return err
			}
		}
	}
	return nil
}

func importProjectRelations(ctx context.Context, tx pgx.Tx, revID int64, projectID ID, source string, sourceRecordID ID, in *ImportProjectInput) error {
	if len(in.Participants) > 0 {
		aID, err := writeProjectAssertion(ctx, tx, revID, projectID, "people", nil, false, &sourceRecordID, nil, nil)
		if err != nil {
			return err
		}
		for _, p := range in.Participants {
			person, err := resolvePersonRef(ctx, tx, p.Ref, source)
			if err != nil {
				continue
			}
			if err := writeProjectPerson(ctx, tx, newID(), projectID, person.ID, aID, p.Role); err != nil {
				return err
			}
		}
	}
	return nil
}

// EachProject returns an iterator over all projects, ordered by id.
func (r *Repo) EachProject(ctx context.Context) iter.Seq2[*Project, error] {
	return r.eachProject(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_projects
		ORDER BY id`)
}

// EachProjectSince returns an iterator over projects updated since the given time, ordered by id.
func (r *Repo) EachProjectSince(ctx context.Context, since time.Time) iter.Seq2[*Project, error] {
	return r.eachProject(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_projects
		WHERE updated_at >= $1
		ORDER BY id`, since)
}

func (r *Repo) eachProject(ctx context.Context, query string, args ...any) iter.Seq2[*Project, error] {
	return func(yield func(*Project, error) bool) {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("eachProject: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanProjectRow(rows)
			if err != nil {
				yield(nil, fmt.Errorf("eachProject: %w", err))
				return
			}
			if !yield(p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("eachProject: %w", err))
		}
	}
}

// scanProject scans a single project row (including cache) from a QueryRow result.
func scanProject(row pgx.Row) (*Project, error) {
	var p Project
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &p.StartDate, &p.EndDate,
		&deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		p.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		p.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		p.DeletedByID = &id
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if err := parseProjectCache(&p, cache); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanProjectRow(row pgx.CollectableRow) (*Project, error) {
	var p Project
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &p.StartDate, &p.EndDate,
		&deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		p.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		p.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		p.DeletedByID = &id
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if err := parseProjectCache(&p, cache); err != nil {
		return nil, err
	}
	return &p, nil
}

func parseProjectCache(p *Project, cache []byte) error {
	if len(cache) == 0 || string(cache) == "{}" {
		return nil
	}
	var d struct {
		Titles       []Title         `json:"titles,omitempty"`
		Descriptions []Text          `json:"descriptions,omitempty"`
		Identifiers  []Identifier    `json:"identifiers,omitempty"`
		People       []ProjectPerson `json:"people,omitempty"`
	}
	if err := json.Unmarshal(cache, &d); err != nil {
		return fmt.Errorf("parseProjectCache: %w", err)
	}
	p.Titles = d.Titles
	p.Descriptions = d.Descriptions
	p.Identifiers = d.Identifiers
	p.People = d.People
	return nil
}
