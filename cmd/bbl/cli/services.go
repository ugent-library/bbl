package cli

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"

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

func newRepo(ctx context.Context) (*pgxrepo.Repo, func(), error) {
	conn, err := pgxpool.New(ctx, config.PgConn)
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
