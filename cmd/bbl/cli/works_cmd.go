package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(worksCmd)
}

var worksCmd = &cobra.Command{
	Use:   "works",
	Short: "Works",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := NewRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.WorksIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}
