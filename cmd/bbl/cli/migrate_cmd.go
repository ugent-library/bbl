package cli

import (
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/pgxrepo"
)

var migrationVersion int

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().IntVar(&migrationVersion, "version", 0, "")
}

var migrateCmd = &cobra.Command{
	Use:       "migrate [up|down]",
	Short:     "Run database migrations",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"up", "down"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == "up" {
			if migrationVersion > 0 {
				return pgxrepo.MigrateUpTo(cmd.Context(), config.PgConn, migrationVersion)
			}
			return pgxrepo.MigrateUp(cmd.Context(), config.PgConn)
		} else {
			if migrationVersion > 0 {
				return pgxrepo.MigrateDownTo(cmd.Context(), config.PgConn, migrationVersion)
			}
			return pgxrepo.MigrateDown(cmd.Context(), config.PgConn)
		}
	},
}
