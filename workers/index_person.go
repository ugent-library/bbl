package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type IndexPerson struct {
	river.WorkerDefaults[jobs.IndexPerson]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewIndexPerson(repo *pgxrepo.Repo, index bbl.Index) *IndexPerson {
	return &IndexPerson{
		repo:  repo,
		index: index,
	}
}

func (w *IndexPerson) Work(ctx context.Context, job *river.Job[jobs.IndexPerson]) error {
	rec, err := w.repo.GetPerson(ctx, job.Args.ID)
	if err != nil {
		return err
	}

	if err := w.index.People().Add(ctx, rec); err != nil {
		return err
	}

	return nil
}
