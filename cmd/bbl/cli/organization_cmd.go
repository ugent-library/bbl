package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
)

func init() {
	rootCmd.AddCommand(organizationCmd)
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.AddCommand(searchOrganizationsCmd)
	searchOrganizationsCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchOrganizationsCmd.Flags().IntVar(&searchOpts.Size, "size", 20, "")
	searchOrganizationsCmd.Flags().IntVar(&searchOpts.From, "from", 0, "")
	searchOrganizationsCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	organizationsCmd.AddCommand(reindexOrganizationsCmd)
}

var organizationCmd = &cobra.Command{
	Use:   "organization [id]",
	Short: "Get organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetOrganization(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return writeData(cmd, rec)
	},
}

var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Organizations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

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
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		info, err := repo.Catbird.RunTaskWait(cmd.Context(),
			tasks.ReindexOrganizationsName,
			tasks.ReindexOrganizationsInput{},
			catbird.RunTaskOpts{DeduplicationID: tasks.ReindexOrganizationsName},
		)
		if err != nil {
			return err
		}

		return writeData(cmd, info)
	},
}

var searchOrganizationsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Organizations().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		return writeData(cmd, hits)
	},
}
