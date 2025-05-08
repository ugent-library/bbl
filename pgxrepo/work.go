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

func (r *Repo) GetWork(ctx context.Context, id string) (*bbl.Work, error) {
	return getWork(ctx, r.conn, id)
}

func (r *Repo) WorksIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Work] {
	q := `
		select id, kind, coalesce(subkind, ''), status, attrs, version, created_at, updated_at, identifiers, contributors, files, rels
		from bbl_works_view;`

	return func(yield func(*bbl.Work) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanWork(rows)
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

func getWork(ctx context.Context, conn pgxConn, id string) (*bbl.Work, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select w.id, w.kind, coalesce(w.subkind, ''), w.status, w.attrs, w.version, w.created_at, w.updated_at, w.identifiers, w.contributors, w.files, w.rels
			from bbl_works_view w, bbl_work_identifiers w_i
			where w.id = w_i.work_id and w_i.scheme = $1 and w_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, kind, coalesce(subkind, ''), status, attrs, version, created_at, updated_at, identifiers, contributors, files, rels
			from bbl_works_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanWork(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetWork %s: %w", id, err)
	}

	return rec, err
}

func scanWork(row pgx.Row) (*bbl.Work, error) {
	var rec bbl.Work
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
	var rawContributors json.RawMessage
	var rawFiles json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rec.Subkind, &rec.Status, &rawAttrs, &rec.Version, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers, &rawContributors, &rawFiles, &rawRels); err != nil {
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

	if rawContributors != nil {
		if err := json.Unmarshal(rawContributors, &rec.Contributors); err != nil {
			return nil, err
		}
	}

	if rawFiles != nil {
		if err := json.Unmarshal(rawFiles, &rec.Files); err != nil {
			return nil, err
		}
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, err
		}
	}

	if err := bbl.LoadWorkProfile(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}
