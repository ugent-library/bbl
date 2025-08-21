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
	q := `select ` + workCols + ` from bbl_works_view w;`

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

func (r *Repo) GetWorkChanges(ctx context.Context, id string) ([]bbl.WorkChange, error) {
	q := `
		select c.rev_id, r.created_at, r.user_id, row_to_json(u) as user, c.diff
		from bbl_changes c
		left join bbl_revs r on r.id = c.rev_id
		left join bbl_users_view u on u.id = r.user_id
		where c.work_id = $1
		order by c.id desc;`

	var changes []bbl.WorkChange

	rows, err := r.conn.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c bbl.WorkChange
		var rawUser json.RawMessage
		var rawDiff json.RawMessage
		err := rows.Scan(&c.RevID, &c.CreatedAt, &c.UserID, &rawUser, &rawDiff)
		if err != nil {
			return nil, err
		}
		if rawUser != nil {
			if err := json.Unmarshal(rawUser, &c.User); err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(rawDiff, &c.Diff); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}

	return changes, nil
}

func getWork(ctx context.Context, conn pgxConn, id string) (*bbl.Work, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select `+workCols+` 
			from bbl_works_view w, bbl_work_identifiers w_i
			where w.id = w_i.work_id and
			      w_i.scheme = $1 and
				  w_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `select `+workCols+` from bbl_works_view w where w.id = $1;`, id)
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

const workCols = `
	w.id,
	w.version,
	w.created_at,
	w.updated_at,
	coalesce(w.created_by_id::text, ''),
	coalesce(w.updated_by_id::text, ''),
	w.created_by,
	w.updated_by,
	w.permissions,
	w.kind,
	coalesce(w.subkind, ''),
	w.status,
	w.attrs,
	w.identifiers,
	w.contributors,
	w.files,
	w.rels
`

func scanWork(row pgx.Row) (*bbl.Work, error) {
	var rec bbl.Work
	var rawCreatedBy json.RawMessage
	var rawUpdatedBy json.RawMessage
	var rawPermissions json.RawMessage
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
	var rawContributors json.RawMessage
	var rawFiles json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(
		&rec.ID,
		&rec.Version,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.CreatedByID,
		&rec.UpdatedByID,
		&rawCreatedBy,
		&rawUpdatedBy,
		&rawPermissions,
		&rec.Kind,
		&rec.Subkind,
		&rec.Status,
		&rawAttrs,
		&rawIdentifiers,
		&rawContributors,
		&rawFiles,
		&rawRels,
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
	if rawPermissions != nil {
		if err := json.Unmarshal(rawPermissions, &rec.Permissions); err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal(rawAttrs, &rec.WorkAttrs); err != nil {
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
