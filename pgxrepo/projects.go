package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) GetProject(ctx context.Context, id string) (*bbl.Project, error) {
	return getProject(ctx, r.conn, id)
}

func (r *Repo) ProjectsIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Project] {
	q := `SELECT ` + projectCols + ` FROM bbl_projects_view p;`

	return func(yield func(*bbl.Project) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanProject(rows)
			if err != nil {
				*errPtr = err
				return
			}
			if !yield(rec) {
				return
			}
		}
	}
}

func getProject(ctx context.Context, conn Conn, id string) (*bbl.Project, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			SELECT `+personCols+`
			FROM bbl_projects_view p, bbl_project_identifiers p_i
			WHERE p.id = p_i.project_id AND p_i.scheme = $1 AND p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `SELECT `+projectCols+` FROM bbl_projects_view p WHERE p.id = $1;`, id)
	}

	rec, err := scanProject(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetProject %s: %w", id, err)
	}

	return rec, err
}

const projectCols = `
	p.id,
	p.version,
	p.created_at,
	p.updated_at,
	coalesce(p.created_by_id::text, ''),
	coalesce(p.updated_by_id::text, ''),
	p.created_by,
	p.updated_by,
	p.attrs,
	p.identifiers
`

func scanProject(row pgx.Row) (*bbl.Project, error) {
	var rec bbl.Project
	var rawCreatedBy json.RawMessage
	var rawUpdatedBy json.RawMessage
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage

	if err := row.Scan(
		&rec.ID,
		&rec.Version,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.CreatedByID,
		&rec.UpdatedByID,
		&rawCreatedBy,
		&rawUpdatedBy,
		&rawAttrs,
		&rawIdentifiers,
	); err != nil {
		return nil, err
	}

	if rawCreatedBy != nil {
		if err := json.Unmarshal(rawCreatedBy, &rec.CreatedBy); err != nil {
			return nil, err
		}
	}
	if rawUpdatedBy != nil {
		if err := json.Unmarshal(rawUpdatedBy, &rec.UpdatedBy); err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal(rawAttrs, &rec.ProjectAttrs); err != nil {
		return nil, err
	}
	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
