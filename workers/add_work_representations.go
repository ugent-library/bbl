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
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewAddWorkRepresentations(repo *pgxrepo.Repo, index bbl.Index) *AddWorkRepresentations {
	return &AddWorkRepresentations{
		repo:  repo,
		index: index,
	}
}

func (w *AddWorkRepresentations) Work(ctx context.Context, job *river.Job[jobs.AddWorkRepresentations]) error {
	rec, err := w.repo.GetWork(ctx, job.Args.ID)
	if err != nil {
		return err
	}
	// TODO schemes hardcoded for now
	for _, scheme := range []string{"oai_dc"} {
		b, err := bbl.EncodeWork(rec, scheme)
		if err != nil {
			return err
		}
		if err = w.repo.AddWorkRepresentation(ctx, rec.ID, scheme, b); err != nil {
			return err
		}
	}
	return nil
}
