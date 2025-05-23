package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(fileCmd)
}

var fileCmd = &cobra.Command{
	Use:   "file [id]",
	Short: "Get file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := newStore()
		if err != nil {
			return err
		}

		return store.Download(cmd.Context(), args[0], cmd.OutOrStdout())
	},
}
