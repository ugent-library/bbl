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
	q := `
		select id, attrs, version, created_at, updated_at, identifiers
		from bbl_projects_view;`

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

func getProject(ctx context.Context, conn pgxConn, id string) (*bbl.Project, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select p.id, p.attrs, p.version, p.created_at, p.updated_at, p.identifiers
			from bbl_projects_view p, bbl_projects_identifiers p_i
			where p.id = p_i.project_id and p_i.scheme = $1 and p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, attrs, version, created_at, updated_at, identifiers
			from bbl_projects_view
			where id = $1;`,
			id,
		)
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

func scanProject(row pgx.Row) (*bbl.Project, error) {
	var rec bbl.Project
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rawAttrs, &rec.Version, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
