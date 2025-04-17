package cli

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/jobs"
)

func init() {
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.AddCommand(searchOrganizationsCmd)
	searchOrganizationsCmd.Flags().IntVar(&searchOpts.Limit, "limit", 20, "")
	searchOrganizationsCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchOrganizationsCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	organizationCmd.AddCommand(reindexOrganizationsCmd)
}

var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Organizations",
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

		for rec := range repo.OrganizationsIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var reindexOrganizationsCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex organizations job",
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

		res, err := riverClient.Insert(cmd.Context(), jobs.ReindexOrganizationsArgs{}, nil)
		if err != nil {
			return err
		}

		if res.UniqueSkippedAsDuplicate {
			logger.Info("organizations reindexer is already running")
		} else {
			logger.Info("started organizations reindexer", "job", res.Job.ID)
		}

		return nil
	},
}

var searchOrganizationsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Organizations().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
