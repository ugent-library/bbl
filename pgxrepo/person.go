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

func (r *Repo) GetPerson(ctx context.Context, id string) (*bbl.Person, error) {
	return getPerson(ctx, r.conn, id)
}

func (r *Repo) PeopleIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Person] {
	q := `
		select id, attrs, created_at, updated_at, identifiers
		from bbl_people_view;`

	return func(yield func(*bbl.Person) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanPerson(rows)
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
func getPerson(ctx context.Context, conn pgxConn, id string) (*bbl.Person, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select p.id, p.attrs, p.created_at, p.updated_at, p.identifiers
			from bbl_people_view w, bbl_people_identifiers p_i
			where p.id = p_i.person_id and p_i.scheme = $1 and p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, attrs, created_at, updated_at, identifiers
			from bbl_people_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanPerson(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetPerson %s: %w", id, err)
	}

	return rec, err
}

func scanPerson(row pgx.Row) (*bbl.Person, error) {
	var rec bbl.Person
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers); err != nil {
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
