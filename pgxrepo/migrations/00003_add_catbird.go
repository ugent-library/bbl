package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/ugent-library/catbird"
)

func init() {
	goose.AddMigrationNoTxContext(addCatbirdUp, addCatbirdDown)
}

func addCatbirdUp(ctx context.Context, db *sql.DB) error {
	return catbird.MigrateUpTo(ctx, db, 2)
}

func addCatbirdDown(ctx context.Context, db *sql.DB) error {
	return catbird.MigrateDownTo(ctx, db, 0)
}
