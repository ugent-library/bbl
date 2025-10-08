package workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportUserSourceInput struct {
	Source string `json:"source"`
}

type ImportUserSourceOutput struct {
	Imported int `json:"imported"`
}

func ImportUserSource(client *hatchet.Client, repo *pgxrepo.Repo) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("import_user_source", func(ctx hatchet.Context, input ImportUserSourceInput) (ImportUserSourceOutput, error) {
		us := bbl.GetUserSource(input.Source)

		seq, finish := us.Iter(ctx)

		out := ImportUserSourceOutput{}

		for rec := range seq {
			if err := repo.SaveUser(ctx, rec); err != nil {
				return out, err
			} else {
				out.Imported++
			}
		}

		return out, finish()
	},
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.source",
			LimitStrategy: &strategyCancelNewest,
		}),
	)
}
