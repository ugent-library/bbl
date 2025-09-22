package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportUserSource struct {
	river.WorkerDefaults[jobs.ImportUserSource]
	repo *pgxrepo.Repo
}

func NewImportUserSource(repo *pgxrepo.Repo) *ImportUserSource {
	return &ImportUserSource{
		repo: repo,
	}
}

func (w *ImportUserSource) Work(ctx context.Context, job *river.Job[jobs.ImportUserSource]) error {
	us := bbl.GetUserSource(job.Args.Name)

	seq, finish := us.Iter(ctx)

	for rec := range seq {
		if err := w.repo.SaveUser(ctx, rec); err != nil {
			return err
		}
	}

	return finish()
}
