package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type IndexOrganization struct {
	river.WorkerDefaults[jobs.IndexOrganization]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewIndexOrganization(repo *pgxrepo.Repo, index bbl.Index) *IndexOrganization {
	return &IndexOrganization{
		repo:  repo,
		index: index,
	}
}

func (w *IndexOrganization) Work(ctx context.Context, job *river.Job[jobs.IndexOrganization]) error {
	rec, err := w.repo.GetOrganization(ctx, job.Args.ID)
	if err != nil {
		return err
	}

	if err := w.index.Organizations().Add(ctx, rec); err != nil {
		return err
	}

	return nil
}
