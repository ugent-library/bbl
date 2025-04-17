package cli

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/oaidcscheme"
	"github.com/ugent-library/bbl/opensearchindex"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/tonga"
)

func NewLogger(w io.Writer) *slog.Logger {
	if config.Env == "development" {
		return slog.New(tint.NewHandler(w, &tint.Options{Level: slog.LevelInfo}))
	} else {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}

func NewRepo(ctx context.Context, conn *pgxpool.Pool) (*pgxrepo.Repo, error) {
	repo, err := pgxrepo.New(ctx, conn)
	if err != nil {
		return nil, err
	}

	if err := repo.Queue().CreateChannel(ctx, "organizations_indexer", "organization", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "people_indexer", "person", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "projects_indexer", "project", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "works_indexer", "work", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "works_indexer", "work", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}
	if err := repo.Queue().CreateChannel(ctx, "works_representations_adder", "work", tonga.ChannelOpts{}); err != nil {
		return nil, err
	}

	return repo, nil
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

func NewRiverClient(logger *slog.Logger, conn *pgxpool.Pool, repo *pgxrepo.Repo, index bbl.Index) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, jobs.NewQueueGcWorker(repo.Queue())); err != nil {
		return nil, err
	}
	if err := river.AddWorkerSafely(workers, jobs.NewReindexPeopleWorker(repo, index)); err != nil {
		return nil, err
	}
	if err := river.AddWorkerSafely(workers, jobs.NewReindexOrganizationsWorker(repo, index)); err != nil {
		return nil, err
	}

	riverClient, err := river.NewClient(riverpgxv5.New(conn), &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(10*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.QueueGcArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
		},
		Workers: workers,
	})
	if err != nil {
		return nil, err
	}

	return riverClient, nil
}

func WorkEncoders() map[string]bbl.WorkEncoder {
	return map[string]bbl.WorkEncoder{
		"oai_dc": oaidcscheme.EncodeWork,
	}
}
