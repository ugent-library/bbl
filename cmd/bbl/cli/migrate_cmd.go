package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newMigrateCmd(cfg *config) *cobra.Command {
	var version int

	cmd := &cobra.Command{
		Use:       "migrate [up|down]",
		Short:     "Run database migrations",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"up", "down"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Conn == "" {
				return errors.New("database connection string required (--conn or $BBL_CONN)")
			}
			ctx := cmd.Context()
			if args[0] == "up" {
				if version > 0 {
					return bbl.MigrateUpTo(ctx, cfg.Conn, version)
				}
				return bbl.MigrateUp(ctx, cfg.Conn)
			}
			if version > 0 {
				return bbl.MigrateDownTo(ctx, cfg.Conn, version)
			}
			return bbl.MigrateDown(ctx, cfg.Conn)
		},
	}

	cmd.Flags().IntVar(&version, "version", 0, "target migration version (default: latest)")

	return cmd
}
