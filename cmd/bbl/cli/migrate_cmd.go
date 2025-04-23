package cli

import (
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/pgxrepo"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:       "migrate [up|down]",
	Short:     "Run database migrations",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"up", "down"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == "up" {
			return pgxrepo.MigrateUp(cmd.Context(), config.PgConn)
		} else {
			return pgxrepo.MigrateDown(cmd.Context(), config.PgConn)
		}
	},
}
