package workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ImportWorkSourceInput struct {
	Source string `json:"source"`
}

type ImportWorkSourceOutput struct {
	Imported int `json:"imported"`
}

func ImportWorkSource(client *hatchet.Client, repo *pgxrepo.Repo) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("import_work_source", func(ctx hatchet.Context, input ImportWorkSourceInput) (ImportWorkSourceOutput, error) {
		ws := bbl.GetWorkSource(input.Source)

		seq, finish := ws.Iter(ctx)

		out := ImportWorkSourceOutput{}

		for rec := range seq {
			if err := bbl.LoadWorkProfile(rec); err != nil {
				return out, err
			}

			dup := false
			for _, iden := range rec.Identifiers {
				if iden.Scheme == ws.MatchIdentifierScheme() {
					if _, err := repo.GetWork(ctx, iden.String()); err == nil {
						dup = true
						break
					}
				}
			}
			if !dup {
				rev := &bbl.Rev{}
				rev.Add(&bbl.SaveWork{Work: rec})
				if err := repo.AddRev(ctx, rev); err != nil {
					return out, err
				} else {
					out.Imported++

				}
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
