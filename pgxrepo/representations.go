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

func (r *Repo) HasRepresentation(ctx context.Context, id, scheme string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM bbl_representations WHERE work_id = $1 AND SCHEME = $2);`

	var exists bool
	if err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&exists); err != nil {
		return false, fmt.Errorf("HasRepresentation: %w", err)
	}
	return exists, nil
}

func (r *Repo) GetRepresentation(ctx context.Context, id, scheme string) (*bbl.Representation, error) {
	q := `SELECT record, updated_at FROM bbl_representations WHERE work_id = $1 AND scheme = $2;`

	repr := bbl.Representation{WorkID: id, Scheme: scheme}

	err := r.conn.QueryRow(ctx, q, id, scheme).Scan(&repr.Record, &repr.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("GetRepresentation: %w", bbl.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("GetRepresentation: %w", err)
	}

	return &repr, nil
}

func getRepresentationsQuery(opts bbl.GetRepresentationsOpts) *sqlbuilder.SelectBuilder {
	b := sqlbuilder.PostgreSQL.NewSelectBuilder()
	b.Select("work_id", "scheme", "record", "updated_at").
		From("bbl_representations").
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
			INSERT INTO bbl_sets (id, name)
			VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING;`,
			bbl.NewID(), set,
		)
	}

	batch.Queue(`
		WITH repr AS (	
			INSERT INTO bbl_representations (id, work_id, scheme, record, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (work_id, scheme) DO UPDATE
			SET record = EXCLUDED.record,
				updated_at = EXCLUDED.updated_at
			RETURNING id
		), sets AS (
	  		SELECT id FROM bbl_sets where name = any($5)
		), del_set_reprs AS (
			DELETE FROM bbl_set_representations
			USING repr, sets
			WHERE representation_id = repr.id AND set_id NOT IN (SELECT id FROM sets)
		)
		INSERT INTO bbl_set_representations (set_id, representation_id)
	  	SELECT sets.id, repr.id 
	  	FROM sets, repr
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
