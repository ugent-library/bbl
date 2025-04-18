package cli

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/jobs"
)

func init() {
	rootCmd.AddCommand(peopleCmd)
	peopleCmd.AddCommand(searchPeopleCmd)
	searchPeopleCmd.Flags().IntVar(&searchOpts.Limit, "limit", 20, "")
	searchPeopleCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchPeopleCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	peopleCmd.AddCommand(reindexPeopleCmd)
}

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "People",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := NewRepo(cmd.Context(), conn)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.PeopleIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var reindexPeopleCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex people job",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := NewLogger(cmd.OutOrStdout())

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := NewRepo(cmd.Context(), conn)
		if err != nil {
			return err
		}

		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		riverClient, err := NewRiverClient(logger, conn, repo, index)
		if err != nil {
			return err
		}

		res, err := riverClient.Insert(cmd.Context(), jobs.ReindexPeopleArgs{}, nil)
		if err != nil {
			return err
		}

		if res.UniqueSkippedAsDuplicate {
			logger.Info("people reindexer is already running")
		} else {
			logger.Info("started people reindexer", "job", res.Job.ID)
		}

		return nil
	},
}

var searchPeopleCmd = &cobra.Command{
	Use:   "search",
	Short: "Search people",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.People().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
