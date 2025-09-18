package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportWorkSource struct {
	river.WorkerDefaults[jobs.ImportWorkSource]
	repo *pgxrepo.Repo
}

func NewImportWorkSource(repo *pgxrepo.Repo) *ImportWorkSource {
	return &ImportWorkSource{
		repo: repo,
	}
}

func (w *ImportWorkSource) Work(ctx context.Context, job *river.Job[jobs.ImportWorkSource]) error {
	ws := bbl.GetWorkSource(job.Args.Name)

	seq, finish := ws.Iter(ctx)

	for rec := range seq {
		dup := false
		for _, iden := range rec.Identifiers {
			if iden.Scheme == ws.MatchIdentifierScheme() {
				if _, err := w.repo.GetWork(ctx, iden.String()); err == nil {
					dup = true
					break
				}
			}
		}
		if !dup {
			rev := &bbl.Rev{}
			rev.Add(&bbl.CreateWork{Work: rec})
			if err := w.repo.AddRev(ctx, rev); err != nil {
				return err
			}
		}
	}

	return finish()
}
