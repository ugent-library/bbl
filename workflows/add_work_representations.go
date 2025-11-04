package workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type AddWorkRepresentationsInput struct {
	Payload bbl.RecordChangedPayload `json:"payload"`
}

type AddWorkRepresentationsOutput struct {
}

// TODO only if public, set deleted otherwise
func AddWorkRepresentations(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("add_work_representations", func(ctx hatchet.Context, input AddWorkRepresentationsInput) (AddWorkRepresentationsOutput, error) {
		out := AddWorkRepresentationsOutput{}

		rec, err := repo.GetWork(ctx, input.Payload.ID)
		if err != nil {
			return out, err
		}

		// TODO schemes hardcoded for now
		for _, scheme := range []string{"oai_dc", "mla"} {
			b, err := bbl.EncodeWork(rec, scheme)
			if err != nil {
				return out, err
			}
			if err = repo.AddWorkRepresentation(ctx, rec.ID, scheme, b); err != nil {
				return out, err
			}
		}

		return out, nil
	},
		hatchet.WithWorkflowEvents(bbl.WorkChangedTopic),
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.id",
			LimitStrategy: &strategyCancelNewest,
		}),
	)
}
