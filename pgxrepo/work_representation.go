package pgxrepo

import (
	"context"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) HasWorkRepresentation(ctx context.Context, id, scheme string) (bool, error) {
	q := `
		select exists(
			select 1 from bbl_work_representations
	      	where work_id = $1 and scheme = $2
		);
	`

	var exists bool
	if err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&exists); err != nil {
		return false, fmt.Errorf("HasWorkRepresentation: %w", err)
	}
	return exists, nil
}

func (r *Repo) GetWorkRepresentation(ctx context.Context, id, scheme string) (*bbl.WorkRepresentation, error) {
	q := `
		select record, updated_at from bbl_work_representations
	    where work_id = $1 and scheme = $2;
	`

	repr := bbl.WorkRepresentation{WorkID: id, Scheme: scheme}

	err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&repr.Record, &repr.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetWorkRepresentation: %w", err)
	}

	return &repr, nil
}

// TODO cursor
func (r *Repo) GetWorkRepresentations(ctx context.Context, opts bbl.GetWorkRepresentationsOpts) ([]*bbl.WorkRepresentation, string, error) {
	b := sqlbuilder.PostgreSQL.NewSelectBuilder()
	b.Select("work_id", "scheme", "record", "updated_at").
		From("bbl_work_representations").
		Limit(opts.Limit)
	if opts.WorkID != "" {
		b.Where(b.Equal("work_id", opts.WorkID))
	}
	if opts.Scheme != "" {
		b.Where(b.Equal("scheme", opts.Scheme))
	}
	if !opts.UpdatedAtLTE.IsZero() {
		b.Where(b.LTE("updated_at", opts.UpdatedAtLTE))
	}
	if !opts.UpdatedAtGTE.IsZero() {
		b.Where(b.LTE("updated_at", opts.UpdatedAtGTE))
	}
	q, args := b.Build()

	rows, err := r.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("GetWorkRepresentations: %w", err)
	}
	recs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[bbl.WorkRepresentation])
	if err != nil {
		return nil, "", fmt.Errorf("GetWorkRepresentations: %w", err)
	}

	return recs, "", nil
}

// TODO
func (r *Repo) GetMoreWorkRepresentations(ctx context.Context, cursor string) ([]*bbl.WorkRepresentation, string, error) {
	return nil, "", nil
}

func (r *Repo) AddWorkRepresentation(ctx context.Context, id string, scheme string, record []byte) error {
	q := `
		insert into bbl_work_representations (work_id, scheme, record, updated_at)
	    values ($1, $2, $3, now())
		on conflict (work_id, scheme) do update
		set record = excluded.record,
		    updated_at = excluded.updated_at;
	`

	_, err := r.conn.Exec(ctx, q, id, scheme, record)
	if err != nil {
		return fmt.Errorf("AddWorkRepresentation: %w", err)
	}
	return nil
}
