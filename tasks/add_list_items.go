package tasks

import (
	"context"
	"iter"
	"slices"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const AddListItemsName = "add_list_items"

type AddListItemsInput struct {
	UserID       string               `json:"user_id,omitempty"`
	TargetListID string               `json:"target_list_id,omitempty"`
	WorkIDs      []string             `json:"work_ids,omitempty"`
	ListID       string               `json:"list_id,omitempty"`
	SearchOpts   *bbl.SearchOpts      `json:"search_opts,omitempty"`
	Changers     []bbl.RawWorkChanger `json:"changers,omitempty"`
}

type AddListItemsOutput struct{}

func AddListItems(repo *pgxrepo.Repo, index bbl.Index) *catbird.Task {
	return catbird.NewTask(AddListItemsName, func(ctx context.Context, input AddListItemsInput) (AddListItemsOutput, error) {
		out := AddListItemsOutput{}

		var err error
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

		// TODO make an iter Group function
		var group []string
		for workID := range iter {
			if group == nil {
				group = make([]string, 0, 200)
			}
			if len(group) < 200 {
				group = append(group, workID)
			} else {
				if err = repo.AddListItems(ctx, input.TargetListID, group); err != nil {
					break
				}
				group = nil
			}
		}
		if len(group) > 0 {
			err = repo.AddListItems(ctx, input.TargetListID, group)
		}

		return out, err
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
