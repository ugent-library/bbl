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
	q := `SELECT ` + organizationCols + ` FROM bbl_organizations_view o;`

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

func getOrganization(ctx context.Context, conn Conn, id string) (*bbl.Organization, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			SELECT `+organizationCols+`
			FROM bbl_organizations_view o, bbl_organization_identifiers o_i
			WHERE o.id = o_i.organizatons_id AND 
			      o_i.scheme = $1 AND 
				  o_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `SELECT `+organizationCols+` FROM bbl_organizations_view o WHERE o.id = $1;`, id)
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

const organizationCols = `
	o.id,
	o.version,
	o.created_at,
	o.updated_at,
	coalesce(o.created_by_id::text, ''),
	coalesce(o.updated_by_id::text, ''),
	o.created_by,
	o.updated_by,
	o.kind,
	o.attrs,
	o.identifiers,
	o.rels
`

func scanOrganization(row pgx.Row) (*bbl.Organization, error) {
	var rec bbl.Organization
	var rawCreatedBy json.RawMessage
	var rawUpdatedBy json.RawMessage
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
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
		&rec.Kind,
		&rawAttrs,
		&rawIdentifiers,
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
	if err := json.Unmarshal(rawAttrs, &rec.OrganizationAttrs); err != nil {
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
