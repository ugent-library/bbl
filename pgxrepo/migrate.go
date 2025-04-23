package pgxrepo

import (
	"context"
	"database/sql"
	"embed"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	_ "github.com/ugent-library/bbl/pgxrepo/migrations"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func MigrateUp(ctx context.Context, dsn string) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	return goose.UpContext(ctx, db, "migrations")
}

func MigrateDown(ctx context.Context, dsn string) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	return goose.ResetContext(ctx, db, "migrations")
}
