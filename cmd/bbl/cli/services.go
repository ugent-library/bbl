package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/opensearchindex"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/s3store"
)

func newLogger(w io.Writer) *slog.Logger {
	if config.Env == "development" {
		return slog.New(tint.NewHandler(w, &tint.Options{Level: slog.LevelInfo}))
	} else {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}
func newPool(ctx context.Context, connStr string, minConns, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	cfg.MinConns = minConns
	cfg.MaxConns = maxConns
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgx pool: %w", err)
	}

	return pool, nil
}

func newRepo(ctx context.Context) (*pgxrepo.Repo, func(), error) {
	conn, err := newPool(ctx, config.PgConn, 4, 20)
	if err != nil {
		return nil, nil, err
	}
	repo, err := pgxrepo.New(ctx, conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return repo, conn.Close, nil
}

func newIndex(ctx context.Context) (bbl.Index, error) {
	client, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: opensearch.Config{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Addresses: config.OpenSearch.URL,
			Username:  config.OpenSearch.Username,
			Password:  config.OpenSearch.Password,
		},
	})
	if err != nil {
		return nil, err
	}

	return opensearchindex.New(ctx, client)
}

func newStore() (*s3store.Store, error) {
	store, err := s3store.New(s3store.Config{
		URL:    config.S3.URL,
		Region: config.S3.Region,
		ID:     config.S3.ID,
		Secret: config.S3.Secret,
		Bucket: config.S3.Bucket,
	})
	if err != nil {
		return nil, err
	}
	return store, nil
}
