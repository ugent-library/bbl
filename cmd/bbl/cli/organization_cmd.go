package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(organizationCmd)
}

var organizationCmd = &cobra.Command{
	Use:   "organization [id]",
	Short: "Get organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := NewRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetOrganization(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(rec)
	},
}
