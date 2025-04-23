package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type IndexWork struct {
	river.WorkerDefaults[jobs.IndexWork]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewIndexWork(repo *pgxrepo.Repo, index bbl.Index) *IndexWork {
	return &IndexWork{
		repo:  repo,
		index: index,
	}
}

func (w *IndexWork) Work(ctx context.Context, job *river.Job[jobs.IndexWork]) error {
	rec, err := w.repo.GetWork(ctx, job.Args.ID)
	if err != nil {
		return err
	}

	if err := w.index.Works().Add(ctx, rec); err != nil {
		return err
	}

	return nil
}
