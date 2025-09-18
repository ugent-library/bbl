package cli

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

var rootCmd = &cobra.Command{
	Use: "bbl",
}

var searchOpts = &bbl.SearchOpts{}
var queryFilter string

func reportJobProgress(ctx context.Context, riverClient *river.Client[pgx.Tx], jobID int64, logger *slog.Logger) error {
	for range time.Tick(time.Second * 5) {
		j, err := riverClient.JobGet(ctx, jobID)
		if err != nil {
			return err
		}

		if len(j.Errors) > 0 {
			logger.Error("job progress", "job", j.ID, "state", j.State, "errors", j.Errors)
		} else {
			logger.Info("job progress", "job", j.ID, "state", j.State)
		}

		if j.State == rivertype.JobStateCompleted || j.State == rivertype.JobStateCancelled || j.State == rivertype.JobStateDiscarded {
			return nil
		}
	}
	return nil
}
