package pgxrepo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) HasSet(ctx context.Context, name string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM bbl_sets WHERE name = $1);`

	var exists bool
	if err := r.conn.QueryRow(ctx, q, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("HasSet: %w", err)
	}
	return exists, nil
}

func (r *Repo) GetSets(ctx context.Context) ([]*bbl.Set, error) {
	q := `SELECT name, description FROM bbl_sets;`

	rows, err := r.conn.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("GetSets: %w", err)
	}

	recs, err := pgx.CollectRows(rows, scanSet)
	if err != nil {
		return nil, fmt.Errorf("GetSets: %w", err)
	}

	return recs, nil
}

func scanSet(row pgx.CollectableRow) (*bbl.Set, error) {
	var rec bbl.Set
	var description *string

	if err := row.Scan(
		&rec.Name,
		&description,
	); err != nil {
		return nil, err
	}

	if description != nil {
		rec.Description = *description
	}

	return &rec, nil
}

func (r *Repo) HasRepresentation(ctx context.Context, id, scheme string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM bbl_representations WHERE work_id = $1 AND SCHEME = $2);`

	var exists bool
	if err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&exists); err != nil {
		return false, fmt.Errorf("HasRepresentation: %w", err)
	}
	return exists, nil
}

func (r *Repo) GetRepresentation(ctx context.Context, id, scheme string) (*bbl.Representation, error) {
	q := `SELECT record, updated_at, sets FROM bbl_representations_view WHERE work_id = $1 AND scheme = $2;`

	rec := bbl.Representation{WorkID: id, Scheme: scheme}

	err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&rec.Record, &rec.UpdatedAt, &rec.Sets)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("GetRepresentation: %w", bbl.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("GetRepresentation: %w", err)
	}

	return &rec, nil
}

func getRepresentationsQuery(opts bbl.GetRepresentationsOpts) *sqlbuilder.SelectBuilder {
	b := sqlbuilder.PostgreSQL.NewSelectBuilder()
	b.Select("work_id", "scheme", "record", "updated_at", "sets").
		From("bbl_representations_view").
		Limit(opts.Limit).
		OrderBy("work_id", "scheme").Asc()
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
	return b
}

func (r *Repo) GetRepresentations(ctx context.Context, opts bbl.GetRepresentationsOpts) ([]*bbl.Representation, string, error) {
	b := getRepresentationsQuery(opts)
	q, args := b.Build()

	rows, err := r.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("GetRepresentations: %w", err)
	}
	recs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[bbl.Representation])
	if err != nil {
		return nil, "", fmt.Errorf("GetRepresentations: %w", err)
	}

	cursor, err := encodeRepresentationsCursor(recs, opts)
	if err != nil {
		return nil, "", fmt.Errorf("GetRepresentations: %w", err)
	}

	return recs, cursor, nil
}

func (r *Repo) GetMoreRepresentations(ctx context.Context, cursor string) ([]*bbl.Representation, string, error) {
	c, err := decodeRepresentationsCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	b := getRepresentationsQuery(c.Opts)
	if c.Opts.WorkID == "" {
		b.Where(b.GT("work_id", c.LastWorkID))
	}
	if c.Opts.Scheme == "" {
		b.Where(b.GT("scheme", c.LastScheme))
	}
	q, args := b.Build()

	rows, err := r.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("GetMoreRepresentations: %w", err)
	}
	recs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[bbl.Representation])
	if err != nil {
		return nil, "", fmt.Errorf("GetMoreRepresentations: %w", err)
	}

	cursor, err = encodeRepresentationsCursor(recs, c.Opts)
	if err != nil {
		return nil, "", fmt.Errorf("GetMoreRepresentations: %w", err)
	}

	return recs, cursor, nil
}

func (r *Repo) AddRepresentation(ctx context.Context, workID string, scheme string, record []byte, sets []string) error {
	batch := &pgx.Batch{}
	for _, set := range sets {
		batch.Queue(`
			INSERT INTO bbl_sets (id, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (name) DO NOTHING;`,
			bbl.NewID(), set, set,
		)
	}

	batch.Queue(`
		WITH rep AS (	
			INSERT INTO bbl_representations (id, work_id, scheme, record, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (work_id, scheme) DO UPDATE
			SET record = EXCLUDED.record,
				updated_at = EXCLUDED.updated_at
			RETURNING id
		), sets AS (
	  		SELECT id FROM bbl_sets where name = any($5)
		), del_set_reps AS (
			DELETE FROM bbl_set_representations
			USING rep, sets
			WHERE representation_id = rep.id AND set_id NOT IN (SELECT id FROM sets)
		)
		INSERT INTO bbl_set_representations (set_id, representation_id)
	  	SELECT sets.id, rep.id 
	  	FROM sets, rep
	    ON CONFLICT (set_id, representation_id) DO NOTHING;`,
		bbl.NewID(), workID, scheme, record, sets,
	)

	if err := r.conn.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("AddRepresentation: %w", err)
	}

	return nil
}

type RepresentationsCursor struct {
	LastWorkID string
	LastScheme string
	Opts       bbl.GetRepresentationsOpts
}

// TODO use a more space efficient serialization format (protobuf?)
func encodeRepresentationsCursor(recs []*bbl.Representation, opts bbl.GetRepresentationsOpts) (string, error) {
	if len(recs) == 0 {
		return "", nil
	}

	lastRec := recs[len(recs)-1]

	c, err := json.Marshal(&RepresentationsCursor{
		LastWorkID: lastRec.WorkID,
		LastScheme: lastRec.Scheme,
		Opts:       opts,
	})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(c), nil
}

func decodeRepresentationsCursor(cursor string) (*RepresentationsCursor, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var c RepresentationsCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
