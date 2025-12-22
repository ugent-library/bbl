package tasks

import (
	"context"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const ImportWorkName = "import_work"

type ImportWorkInput struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

type ImportWorkOutput struct {
	WorkID string `json:"work_id"`
}

func ImportWork(repo *pgxrepo.Repo) *catbird.Task {
	return catbird.NewTask(ImportWorkName, func(ctx context.Context, input ImportWorkInput) (ImportWorkOutput, error) {
		out := ImportWorkOutput{}

		rec, err := bbl.ImportWork(input.Source, input.ID)
		if err != nil {
			return out, err
		}

		rev := &bbl.Rev{}
		rev.Add(&bbl.SaveWork{Work: rec})
		if err := repo.AddRev(ctx, rev); err != nil {
			return out, err
		}

		out.WorkID = rec.ID

		return out, nil
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
