package workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type AddRepresentationsInput struct {
	Payload bbl.RecordChangedPayload `json:"payload"`
}

type AddRepresentationsOutput struct {
}

// TODO only if public, set deleted otherwise
func AddRepresentations(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("add_representations", func(ctx hatchet.Context, input AddRepresentationsInput) (AddRepresentationsOutput, error) {
		out := AddRepresentationsOutput{}

		rec, err := repo.GetWork(ctx, input.Payload.ID)
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
		hatchet.WithWorkflowEvents(bbl.WorkChangedTopic),
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.payload.id",
			LimitStrategy: &strategyCancelNewest,
		}),
	)
}
