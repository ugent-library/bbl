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

func (r *Repo) GetOrganization(ctx context.Context, id string) (*bbl.Organization, error) {
	return getOrganization(ctx, r.conn, id)
}

func (r *Repo) OrganizationsIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.Organization] {
	q := `
		select id, kind, attrs, version, created_at, updated_at, identifiers, rels
		from bbl_organizations_view;`

	return func(yield func(*bbl.Organization) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanOrganization(rows)
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

func getOrganization(ctx context.Context, conn pgxConn, id string) (*bbl.Organization, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select o.id, o.kind, o.attrs, o.version, o.created_at, o.updated_at, o.identifiers, o.rels
			from bbl_organizations_view o, bbl_organizations_identifiers o_i
			where o.id = o_i.organizatons_id and o_i.scheme = $1 and o_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, kind, attrs, version, created_at, updated_at, identifiers, rels
			from bbl_organizations_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanOrganization(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetOrganization %s: %w", id, err)
	}

	return rec, err
}

func scanOrganization(row pgx.Row) (*bbl.Organization, error) {
	var rec bbl.Organization
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rawAttrs, &rec.Version, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers, &rawRels); err != nil {
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

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
