package pgxrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ugent-library/bbl"
)

func (r *Repo) GetProject(ctx context.Context, id string) (*bbl.Project, error) {
	return getProject(ctx, r.conn, id)
}

func (r *Repo) ProjectsIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Project] {
	return func(yield func(*bbl.Project) bool) {
		q := `SELECT ` + projectCols + ` FROM bbl_projects_view p;`
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = fmt.Errorf("ProjectsIter: query: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanProject(rows)
			if err != nil {
				*errPtr = fmt.Errorf("ProjectsIter: scan: %w", err)
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
		// query will fail if identifier is not unique
		row = conn.QueryRow(ctx, `
			SELECT `+projectCols+`
			FROM bbl_projects_view p
			WHERE p.id = (SELECT project_id
						  FROM bbl_project_identifiers
						  WHERE scheme = $1 AND val = $2);`,
			scheme, val)
	} else {
		row = conn.QueryRow(ctx, `SELECT `+projectCols+` FROM bbl_projects_view p WHERE p.id = $1;`, id)
	}

	rec, err := scanProject(row)
	if errors.Is(err, pgx.ErrNoRows) {
		err = bbl.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "21000" { // cardinality_violation (mare than one row returned from subquery)
			err = bbl.ErrNotUnique
		}
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
