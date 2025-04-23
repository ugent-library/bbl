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
	"github.com/ugent-library/bbl/workers"
)

func newLogger(w io.Writer) *slog.Logger {
	if config.Env == "development" {
		return slog.New(tint.NewHandler(w, &tint.Options{Level: slog.LevelInfo}))
	} else {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
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

func newRiverClient(logger *slog.Logger, conn *pgxpool.Pool, repo *pgxrepo.Repo, index bbl.Index) (*river.Client[pgx.Tx], error) {
	w := river.NewWorkers()
	river.AddWorker(w, workers.NewQueueGc(repo.Queue()))
	river.AddWorker(w, workers.NewIndexPerson(repo, index))
	river.AddWorker(w, workers.NewReindexPeople(repo, index))
	river.AddWorker(w, workers.NewIndexOrganization(repo, index))
	river.AddWorker(w, workers.NewReindexOrganizations(repo, index))
	river.AddWorker(w, workers.NewIndexProject(repo, index))
	river.AddWorker(w, workers.NewReindexProjects(repo, index))
	river.AddWorker(w, workers.NewAddWorkRepresentations(repo, index, workEncoders()))
	river.AddWorker(w, workers.NewIndexWork(repo, index))
	river.AddWorker(w, workers.NewReindexWorks(repo, index))

	riverClient, err := river.NewClient(riverpgxv5.New(conn), &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(10*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.QueueGc{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
		},
		Workers: w,
	})
	if err != nil {
		return nil, err
	}

	return riverClient, nil
}

func workEncoders() map[string]bbl.WorkEncoder {
	return map[string]bbl.WorkEncoder{
		"oai_dc": oaidcscheme.EncodeWork,
	}
}
