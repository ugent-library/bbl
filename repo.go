package bbl

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Services bundles the core runtime dependencies shared across the CLI and
// the HTTP layer. It lives in this package so that cmd/, app/, and any other
// package can import it without circularity.
type Services struct {
	Repo    *Repo
	Sources map[string]UserSource
}

// Repo is the single repository backed by PostgreSQL.
// No interface is defined — PostgreSQL features are used pervasively
// and there are no plans to support another database.
type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(ctx context.Context, connString string) (*Repo, error) {
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() {
	r.db.Close()
}
