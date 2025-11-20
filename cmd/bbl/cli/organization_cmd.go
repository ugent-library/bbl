package cli

import (
	"encoding/json"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/workflows"
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

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ReindexOrganizations(hatchetClient, repo, index)

		res, err := task.Run(cmd.Context(), workflows.ReindexOrganizationsInput{})
		if err != nil {
			return err
		}

		out := workflows.ReindexOrganizationsOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
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
