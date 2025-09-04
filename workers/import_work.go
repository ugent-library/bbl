package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportWork struct {
	river.WorkerDefaults[jobs.ImportWork]
	repo *pgxrepo.Repo
}

func NewImportWork(repo *pgxrepo.Repo) *ImportWork {
	return &ImportWork{
		repo: repo,
	}
}

func (w *ImportWork) Work(ctx context.Context, job *river.Job[jobs.ImportWork]) error {
	rec, err := bbl.ImportWork(job.Args.Source, job.Args.ID)
	if err != nil {
		return err
	}

	rev := &bbl.Rev{}
	rev.Add(&bbl.CreateWork{Work: rec})
	if err := w.repo.AddRev(ctx, rev); err != nil {
		return err
	}

	out := jobs.ImportWorkOutput{WorkID: rec.ID}
	if err := river.RecordOutput(ctx, &out); err != nil {
		return err
	}

	return nil
}
