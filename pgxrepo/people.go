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
	q := `SELECT ` + personCols + ` FROM bbl_people_view p;`

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

// TODO include in users view?
func (r *Repo) GetPeopleIDsByIdentifiers(ctx context.Context, identifiers []bbl.Code) ([]string, error) {
	var qVals string
	var qVars []any
	for i, iden := range identifiers {
		if i > 0 {
			qVals += `,`
		}
		qVals += fmt.Sprintf(`($%d, $%d)`, len(qVars)+1, len(qVars)+2)
		qVars = append(qVars, iden.Scheme, iden.Val)
	}

	q := `SELECT DISTINCT person_id FROM bbl_person_identifiers WHERE (scheme, val) = any(values ` + qVals + `);`

	rows, err := r.conn.Query(ctx, q, qVars...)
	if err != nil {
		return nil, err
	}

	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func getPerson(ctx context.Context, conn Conn, id string) (*bbl.Person, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			SELECT `+personCols+`
			FROM bbl_people_view p, bbl_person_identifiers p_i
			WHERE p.id = p_i.person_id AND
			      p_i.scheme = $1 AND
				  p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `SELECT `+personCols+` FROM bbl_people_view p WHERE p.id = $1;`, id)
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

const personCols = `
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

func scanPerson(row pgx.Row) (*bbl.Person, error) {
	var rec bbl.Person
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
	if err := json.Unmarshal(rawAttrs, &rec.PersonAttrs); err != nil {
		return nil, err
	}
	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
