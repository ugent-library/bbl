package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type IndexProject struct {
	river.WorkerDefaults[jobs.IndexProject]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewIndexProject(repo *pgxrepo.Repo, index bbl.Index) *IndexProject {
	return &IndexProject{
		repo:  repo,
		index: index,
	}
}

func (w *IndexProject) Work(ctx context.Context, job *river.Job[jobs.IndexProject]) error {
	rec, err := w.repo.GetProject(ctx, job.Args.ID)
	if err != nil {
		return err
	}

	if err := w.index.Projects().Add(ctx, rec); err != nil {
		return err
	}

	return nil
}
