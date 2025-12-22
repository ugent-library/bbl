package tasks

import (
	"context"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const AddRepresentationsName = "add_representations"

type AddRepresentationsInput struct {
	WorkID string `json:"work_id"`
}

type AddRepresentationsOutput struct {
}

// TODO only if public, set deleted otherwise
func AddRepresentations(repo *pgxrepo.Repo, index bbl.Index) *catbird.Task {
	return catbird.NewTask(string(AddRepresentationsName), func(ctx context.Context, input AddRepresentationsInput) (AddRepresentationsOutput, error) {
		out := AddRepresentationsOutput{}

		rec, err := repo.GetWork(ctx, input.WorkID)
		if err != nil {
			return out, err
		}

		// TODO schemes and sets hardcoded for now
		for _, scheme := range []string{"oai_dc", "mla"} {
			b, err := bbl.EncodeWork(rec, scheme)
			if err != nil {
				return out, err
			}
			if err = repo.AddRepresentation(ctx, rec.ID, scheme, b, []string{rec.Kind}); err != nil {
				return out, err
			}
		}

		return out, nil
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
