package workflows

import (
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ReindexWorkSearchesInput struct {
}

type ReindexWorkSearchesOutput struct {
}

func ReindexWorkSearches(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("reindex_work_searches", func(ctx hatchet.Context, input ReindexWorkSearchesInput) (ReindexWorkSearchesOutput, error) {
		out := ReindexWorkSearchesOutput{}

		switcher, err := index.WorkSearches().NewSwitcher(ctx)
		if err != nil {
			return out, err
		}

		for rec := range repo.WorkSearchesIter(ctx, &err) {
			if err = switcher.Add(ctx, rec.Query); err != nil {
				return out, err
			}
		}

		if err != nil {
			return out, err
		}

		return out, switcher.Switch(ctx)
	},
		hatchet.WithCron("0 0 * * *"),
	)
}
