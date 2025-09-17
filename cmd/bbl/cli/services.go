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
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/opensearchindex"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/s3store"
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

func newInsertOnlyRiverClient(logger *slog.Logger, conn *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	client, err := river.NewClient(riverpgxv5.New(conn), &river.Config{
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newRiverClient(logger *slog.Logger, conn *pgxpool.Pool, repo *pgxrepo.Repo, index bbl.Index, store *s3store.Store, hub *catbird.Hub) (*river.Client[pgx.Tx], error) {
	w := river.NewWorkers()
	river.AddWorker(w, workers.NewQueueGc(repo.Queue()))
	river.AddWorker(w, workers.NewIndexPerson(repo, index))
	river.AddWorker(w, workers.NewReindexPeople(repo, index))
	river.AddWorker(w, workers.NewIndexOrganization(repo, index))
	river.AddWorker(w, workers.NewReindexOrganizations(repo, index))
	river.AddWorker(w, workers.NewIndexProject(repo, index))
	river.AddWorker(w, workers.NewReindexProjects(repo, index))
	river.AddWorker(w, workers.NewAddWorkRepresentations(repo, index))
	river.AddWorker(w, workers.NewIndexWork(repo, index))
	river.AddWorker(w, workers.NewReindexWorks(repo, index))
	river.AddWorker(w, workers.NewExportWorks(index, store, hub))
	river.AddWorker(w, workers.NewImportWork(repo))
	river.AddWorker(w, workers.NewImportWorkSource(repo))

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(10*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return jobs.QueueGc{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}

	for name := range bbl.WorkSources() {
		ws := bbl.GetWorkSource(name)
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(ws.Interval()),
			func() (river.JobArgs, *river.InsertOpts) {
				return jobs.ImportWorkSource{Name: name}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
	}

	client, err := river.NewClient(riverpgxv5.New(conn), &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		PeriodicJobs: periodicJobs,
		Workers:      w,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
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
