package tasks

import (
	"context"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const ImportWorkSourceName = "import_work_source"

type ImportWorkSourceInput struct {
	Source string `json:"source"`
}

type ImportWorkSourceOutput struct {
	Imported int `json:"imported"`
}

func ImportWorkSource(repo *pgxrepo.Repo) *catbird.Task {
	return catbird.NewTask(ImportWorkSourceName, func(ctx context.Context, input ImportWorkSourceInput) (ImportWorkSourceOutput, error) {
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
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
