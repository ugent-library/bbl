package workflows

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"
	"time"

	"github.com/centrifugal/gocent/v3"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/s3store"
	"golang.org/x/sync/errgroup"
)

type ExportWorksInput struct {
	UserID     string          `json:"user_id,omitempty"`
	WorkIDs    []string        `json:"work_ids,omitempty"`
	ListID     string          `json:"list_id,omitempty"`
	SearchOpts *bbl.SearchOpts `json:"search_opts"`
	Format     string          `json:"format"`
}

type ExportWorksOutput struct {
	FileID string `json:"file_id"`
}

func ExportWorks(client *hatchet.Client, store *s3store.Store, repo *pgxrepo.Repo, index bbl.Index, centrifugeClient *gocent.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("export_works", func(ctx hatchet.Context, input ExportWorksInput) (ExportWorksOutput, error) {
		out := ExportWorksOutput{}

		pr, pw := io.Pipe()

		fileID := bbl.NewID()

		group, groupCtx := errgroup.WithContext(ctx)

		group.Go(func() error { // TODO file expiry
			return store.Upload(groupCtx, fileID, pr)
		})

		group.Go(func() error {
			defer pw.Close()

			exp, err := bbl.NewWorkExporter(pw, input.Format)
			if err != nil {
				return err
			}

			var iter iter.Seq[*bbl.Work]

			if input.WorkIDs != nil {
				iter = func(yield func(*bbl.Work) bool) {
					for _, id := range input.WorkIDs {
						rec, e := repo.GetWork(ctx, id) // TODO get recs in one query
						if e != nil {
							err = e
							return
						}
						if !yield(rec) {
							return
						}
					}
				}
			} else if input.ListID != "" {
				iter = func(yield func(*bbl.Work) bool) {
					for listItem := range repo.ListItemsIter(ctx, input.ListID, &err) {
						if !yield(listItem.Work) {
							return
						}
					}
				}
			} else if input.SearchOpts != nil {
				iter = func(yield func(*bbl.Work) bool) {
					for rec := range bbl.SearchIter(ctx, index.Works(), input.SearchOpts, &err) {
						if !yield(rec) {
							return
						}
					}
				}
			}

			for rec := range iter {
				if err := exp.Add(rec); err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}

			if err := exp.Done(); err != nil {
				return err
			}

			return nil
		})

		if err := group.Wait(); err != nil {
			return out, err
		}

		out.FileID = fileID

		if input.UserID != "" { // TODO rendering is no concern of the worker
			presignedURL, err := store.NewDownloadURL(ctx, fileID, 15*time.Minute)
			if err != nil {
				return out, err
			}

			t := views.AddFlash(views.FlashArgs{
				Type:  views.FlashSuccess,
				Title: "Export ready",
				HTML:  `Your export can be downloaded <a href="` + presignedURL + `">here</a>.`, // TODO no raw html; use templ.Component
			})

			var b strings.Builder
			if err := t.Render(ctx, &b); err != nil {
				return out, err
			}
			data, err := json.Marshal(&struct {
				Content string `json:"content"`
			}{
				Content: b.String(),
			})
			if err != nil {
				return out, err
			}

			if _, err = centrifugeClient.Publish(ctx, "users#"+input.UserID, data); err != nil {
				return out, fmt.Errorf("could not publish to centrifuge: %w", err)
			}
		}
		return out, nil
	},
	)
}
