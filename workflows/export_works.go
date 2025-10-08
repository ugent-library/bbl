package workflows

import (
	"io"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/s3store"
	"golang.org/x/sync/errgroup"
)

type ExportWorksInput struct {
	UserID string          `json:"user_id,omitempty"`
	Opts   *bbl.SearchOpts `json:"opts"`
	Format string          `json:"format"`
}

type ExportWorksOutput struct {
	FileID string `json:"file_id"`
}

func ExportWorks(client *hatchet.Client, store *s3store.Store, index bbl.Index, hub *catbird.Hub) *hatchet.StandaloneTask {
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

			for rec := range bbl.SearchIter(groupCtx, index.Works(), input.Opts, &err) {
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

		if input.UserID != "" { // TODO this is no concern of the worker
			presignedURL, err := store.NewDownloadURL(ctx, fileID, 15*time.Minute)
			if err != nil {
				return out, err
			}

			err = hub.Render(ctx, "users."+input.UserID, "flash", views.Flash(views.FlashArgs{
				Type:  views.FlashSuccess,
				Title: "Export ready",
				HTML:  `Your export can be downloaded <a href="` + presignedURL + `">here</a>.`, // TODO no raw html; use templ.Component
			}))
			if err != nil {
				return out, err
			}
		}

		return out, nil
	},
	)
}
