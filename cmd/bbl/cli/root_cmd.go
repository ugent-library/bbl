package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd(reg Registry) *cobra.Command {
	e := &env{cfg: defaultConfig(), reg: reg}

	root := &cobra.Command{
		Use:           "bbl",
		Short:         "bbl repository CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if e.cfg.ConfigFile != "" {
				return e.cfg.loadFile()
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&e.cfg.Conn, "conn", e.cfg.Conn, "PostgreSQL connection string [$BBL_CONN]")
	root.PersistentFlags().StringVar(&e.cfg.ConfigFile, "config", e.cfg.ConfigFile, "Config file path [$BBL_CONFIG]")

	root.AddCommand(newMigrateCmd(&e.cfg))
	// Future commands that need services are methods on e, e.g.:
	// root.AddCommand(e.newSweepCmd())

	return root
}
