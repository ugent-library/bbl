package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

type AddWorkRepresentations struct {
	river.WorkerDefaults[jobs.AddWorkRepresentations]
	repo         *pgxrepo.Repo
	index        bbl.Index
	workEncoders map[string]bbl.WorkEncoder
}

func NewAddWorkRepresentations(repo *pgxrepo.Repo, index bbl.Index, workEncoders map[string]bbl.WorkEncoder) *AddWorkRepresentations {
	return &AddWorkRepresentations{
		repo:         repo,
		index:        index,
		workEncoders: workEncoders,
	}
}

func (w *AddWorkRepresentations) Work(ctx context.Context, job *river.Job[jobs.AddWorkRepresentations]) error {
	rec, err := w.repo.GetWork(ctx, job.Args.ID)
	if err != nil {
		return err
	}
	for scheme, enc := range w.workEncoders {
		b, err := enc(rec)
		if err != nil {
			return err
		}
		if err = w.repo.AddWorkRepresentation(ctx, rec.ID, scheme, b); err != nil {
			return err
		}
	}
	return nil
}
