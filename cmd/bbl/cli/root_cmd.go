package cli

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"github.com/ugent-library/bbl"
)

var (
	prettify    = false
	searchOpts  = &bbl.SearchOpts{}
	queryFilter = ""
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&prettify, "pretty", false, "")
}

var rootCmd = &cobra.Command{
	Use: "bbl",
}

func writeData(cmd *cobra.Command, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return writeJSON(cmd, b)
}

func writeJSON(cmd *cobra.Command, b []byte) error {
	if prettify {
		b = pretty.Color(pretty.Pretty(b), pretty.TerminalStyle)
	}
	_, err := cmd.OutOrStdout().Write(b)
	return err
}

func reportJobProgress(ctx context.Context, riverClient *river.Client[pgx.Tx], jobID int64, logger *slog.Logger) error {
	for range time.Tick(time.Second * 3) {
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
