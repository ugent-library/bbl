package tasks

import (
	"context"
	"log/slog"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const ImportUserSourceName = "import_user_source"

type ImportUserSourceInput struct {
	Source string `json:"source"`
}

type ImportUserSourceOutput struct {
	Imported int `json:"imported"`
	Failed   int `json:"failed"`
}

func ImportUserSource(repo *pgxrepo.Repo, log *slog.Logger) *catbird.Task {
	return catbird.NewTask(ImportUserSourceName, func(ctx context.Context, input ImportUserSourceInput) (ImportUserSourceOutput, error) {
		us := bbl.GetUserSource(input.Source)

		var err error

		seq := us.Iter(ctx, &err)

		out := ImportUserSourceOutput{}

		for rec := range seq {
			rev := &bbl.Rev{}
			rev.Add(&bbl.SaveUser{
				User:         rec,
				MatchVersion: false,
			})

			if err := repo.AddRev(ctx, rev); err != nil {
				log.ErrorContext(ctx, err.Error())
				out.Failed++
			} else {
				out.Imported++
			}
		}

		return out, err
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
