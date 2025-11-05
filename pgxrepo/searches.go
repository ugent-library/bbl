package pgxrepo

import (
	"context"
	"fmt"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) OrganizationSearchesIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Search] {
	return r.searchesIter(ctx, "organization", errPtr)
}

func (r *Repo) PersonSearchesIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Search] {
	return r.searchesIter(ctx, "person", errPtr)
}

func (r *Repo) ProjectSearchesIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Search] {
	return r.searchesIter(ctx, "project", errPtr)
}

func (r *Repo) WorkSearchesIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Search] {
	return r.searchesIter(ctx, "work", errPtr)
}

// TODO gather these in memory first and periodically save them
// or add to unlogged table first and periodically move them

func (r *Repo) AddOrganizationSearch(ctx context.Context, query string) error {
	return r.addSearch(ctx, "AddOrganizationSearch", "organization", query)
}

func (r *Repo) AddPersonSearch(ctx context.Context, query string) error {
	return r.addSearch(ctx, "AddPersonSearch", "person", query)
}

func (r *Repo) AddProjectSearch(ctx context.Context, query string) error {
	return r.addSearch(ctx, "AddProjectSearch", "project", query)
}

func (r *Repo) AddWorkSearch(ctx context.Context, query string) error {
	return r.addSearch(ctx, "AddWorkSearch", "work", query)
}

func (r *Repo) addSearch(ctx context.Context, meth, kind, query string) error {
	q := `
		INSERT INTO bbl_` + kind + `_searches AS s (query, total)
	    VALUES ($1, 1)
		ON CONFLICT (query) DO UPDATE
		SET total = s.total + 1;
	`

	_, err := r.conn.Exec(ctx, q, query)
	if err != nil {
		return fmt.Errorf("%s: %w", meth, err)
	}
	return nil
}

func (r *Repo) searchesIter(ctx context.Context, kind string, errPtr *error) iter.Seq[*bbl.Search] {
	q := `select ` + searchCols + ` from bbl_` + kind + `_searches s;`

	return func(yield func(*bbl.Search) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanSearch(rows)
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

const searchCols = `
	s.query,
	s.total
`

func scanSearch(row pgx.Row) (*bbl.Search, error) {
	var rec bbl.Search

	if err := row.Scan(&rec.Query, &rec.Total); err != nil {
		return nil, err
	}

	return &rec, nil
}
