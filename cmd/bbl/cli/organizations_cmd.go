package cli

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.AddCommand(searchOrganizationsCmd)
	searchOrganizationsCmd.Flags().IntVar(&searchArgs.Limit, "limit", 20, "")
	searchOrganizationsCmd.Flags().StringVarP(&searchArgs.Query, "query", "q", "", "")
	searchOrganizationsCmd.Flags().StringVar(&searchArgs.Cursor, "cursor", "", "")
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

var searchOrganizationsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Organizations().Search(cmd.Context(), searchArgs)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
