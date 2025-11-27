package workflows

import (
	"iter"
	"slices"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ChangeWorksInput struct {
	UserID     string               `json:"user_id,omitempty"`
	WorkIDs    []string             `json:"work_ids,omitempty"`
	ListID     string               `json:"list_id,omitempty"`
	SearchOpts *bbl.SearchOpts      `json:"search_opts,omitempty"`
	Changers   []bbl.RawWorkChanger `json:"changers,omitempty"`
}

type ChangeWorksOutput struct {
	Changed int `json:"changed"`
	Failed  int `json:"failed"`
}

func ChangeWorks(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("change_works", func(ctx hatchet.Context, input ChangeWorksInput) (ChangeWorksOutput, error) {
		out := ChangeWorksOutput{}

		changers, err := bbl.LoadWorkChangers(input.Changers)
		if err != nil {
			return out, err
		}

		var iter iter.Seq[string]

		if input.WorkIDs != nil {
			iter = slices.Values(input.WorkIDs)
		} else if input.ListID != "" { // TODO this loads too much data, only the id is needed
			iter = func(yield func(string) bool) {
				for listItem := range repo.ListItemsIter(ctx, input.ListID, &err) {
					if !yield(listItem.WorkID) {
						return
					}
				}
			}
		} else if input.SearchOpts != nil { // TODO this loads too much data, only the id is needed
			iter = func(yield func(string) bool) {
				for rec := range bbl.SearchIter(ctx, index.Works(), input.SearchOpts, &err) {
					if !yield(rec.ID) {
						return
					}
				}
			}
		}

		for workID := range iter {
			rev := &bbl.Rev{UserID: input.UserID}
			rev.Add(&bbl.ChangeWork{WorkID: workID, Changes: changers})
			if err := repo.AddRev(ctx, rev); err != nil {
				ctx.Log(err.Error()) // TODO
				out.Failed++
			} else {
				out.Changed++
			}
		}

		return out, err
	},
	)
}
