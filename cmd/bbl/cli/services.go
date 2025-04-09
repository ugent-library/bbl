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
	"github.com/ugent-library/tonga"
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

	repo, err := bbl.NewRepo(ctx, conn)
	if err != nil {
		return nil, nil, err
	}

	if err := repo.Queue().CreateChannel(ctx, "organizations_indexer", "organization", tonga.ChannelOpts{}); err != nil {
		return nil, nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "people_indexer", "person", tonga.ChannelOpts{}); err != nil {
		return nil, nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "projects_indexer", "project", tonga.ChannelOpts{}); err != nil {
		return nil, nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "works_indexer", "work", tonga.ChannelOpts{}); err != nil {
		return nil, nil, err
	}

	return repo, conn.Close, nil
}

func NewIndex(ctx context.Context) (bbl.Index, error) {
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
