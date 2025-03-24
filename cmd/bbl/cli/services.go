package cli

import (
	"context"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/ugent-library/bbl"
)

func NewLogger(w io.Writer) *slog.Logger {
	if config.Env == "development" {
		return slog.New(tint.NewHandler(w, &tint.Options{Level: slog.LevelDebug}))
	} else {
		return slog.New(slog.NewJSONHandler(w, nil))
	}
}

func NewRepo(ctx context.Context) (*bbl.Repo, func(), error) {
	conn, err := pgxpool.New(ctx, config.PgConn)
	if err != nil {
		return nil, nil, err
	}
	repo, err := bbl.NewRepo(conn)
	if err != nil {
		return nil, nil, err
	}
	return repo, conn.Close, nil
}
