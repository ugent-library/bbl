package workflows

import (
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportWorkInput struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

type ImportWorkOutput struct {
	WorkID string `json:"work_id"`
}

func ImportWork(client *hatchet.Client, repo *pgxrepo.Repo) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("import_work", func(ctx hatchet.Context, input ImportWorkInput) (ImportWorkOutput, error) {
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
	)
}
