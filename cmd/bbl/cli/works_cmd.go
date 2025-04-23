package cli

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
)

func init() {
	rootCmd.AddCommand(worksCmd)
	worksCmd.AddCommand(searchWorksCmd)
	searchWorksCmd.Flags().IntVar(&searchOpts.Limit, "limit", 20, "")
	searchWorksCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchWorksCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	worksCmd.AddCommand(reindexWorksCmd)
}

var worksCmd = &cobra.Command{
	Use:   "works",
	Short: "Works",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.WorksIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var reindexWorksCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex works job",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := newLogger(cmd.OutOrStdout())

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		riverClient, err := newRiverClient(logger, conn, repo, index)
		if err != nil {
			return err
		}

		res, err := riverClient.Insert(cmd.Context(), jobs.ReindexWorks{}, nil)
		if err != nil {
			return err
		}

		if res.UniqueSkippedAsDuplicate {
			logger.Info("works reindexer is already running")
		} else {
			logger.Info("started works reindexer", "job", res.Job.ID)
		}

		return nil
	},
}

var searchWorksCmd = &cobra.Command{
	Use:   "search",
	Short: "Search works",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Works().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
