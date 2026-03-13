package bbl

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo is the single repository backed by PostgreSQL.
// No interface is defined — PostgreSQL features are used pervasively
// and there are no plans to support another database.
type Repo struct {
	db           *pgxpool.Pool
	tokenKey     []byte        // 32-byte AES-256-GCM key for encrypting user tokens
	WorkProfiles *WorkProfiles // nil = no profile validation
}

func NewRepo(ctx context.Context, connString string, tokenKey []byte) (*Repo, error) {
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &Repo{db: db, tokenKey: tokenKey}, nil
}

func (r *Repo) Close() {
	r.db.Close()
}
