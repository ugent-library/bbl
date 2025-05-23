package workers

import (
	"context"
	"io"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/s3store"
	"github.com/ugent-library/bbl/jobs"
	"golang.org/x/sync/errgroup"
)

type ExportWorks struct {
	river.WorkerDefaults[jobs.ExportWorks]
	index bbl.Index
	store *s3store.Store
}

func NewExportWorks(index bbl.Index, store *s3store.Store) *ExportWorks {
	return &ExportWorks{
		index: index,
		store: store,
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

	return nil
}
