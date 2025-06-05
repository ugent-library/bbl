package workers

import (
	"context"
	"io"
	"time"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/s3store"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/jobs"
	"golang.org/x/sync/errgroup"
)

type ExportWorks struct {
	river.WorkerDefaults[jobs.ExportWorks]
	index bbl.Index
	store *s3store.Store
	hub   *catbird.Hub
}

func NewExportWorks(index bbl.Index, store *s3store.Store, hub *catbird.Hub) *ExportWorks {
	return &ExportWorks{
		index: index,
		store: store,
		hub:   hub,
	}
}

func (w *ExportWorks) Work(ctx context.Context, job *river.Job[jobs.ExportWorks]) error {
	pr, pw := io.Pipe()

	fileID := bbl.NewID()

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error { // TODO file expiry
		return w.store.Upload(groupCtx, fileID, pr)
	})

	group.Go(func() error {
		defer pw.Close()

		exp, err := bbl.NewWorkExporter(pw, job.Args.Format)
		if err != nil {
			return err
		}

		for rec := range bbl.SearchIter(groupCtx, w.index.Works(), job.Args.Opts, &err) {
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
		return err
	}

	out := jobs.ExportWorksOutput{FileID: fileID}
	if err := river.RecordOutput(ctx, &out); err != nil {
		return err
	}

	if job.Args.UserID != "" { // TODO this is no concern of the worker
		presignedURL, err := w.store.NewDownloadURL(ctx, fileID, 15*time.Minute)
		if err != nil {
			return err
		}

		err = w.hub.Render(ctx, "users."+job.Args.UserID, "flash", views.Flash(views.FlashArgs{
			Type:  views.FlashSuccess,
			Title: "Export ready",
			HTML:  `Your export can be downloaded <a href="` + presignedURL + `">here</a>.`, // TODO no raw html; use templ.Component
		}))
		if err != nil {
			return err
		}
	}

	return nil
}
