package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func NewRootCmd(reg *Registry) *cobra.Command {
	cfgFile := os.Getenv("BBL_CONFIG")
	cfg := &config{}
	e := &env{cfg: cfg, reg: reg}

	root := &cobra.Command{
		Use:           "bbl",
		Short:         "bbl repository CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgFile != "" {
				if err := loadConfig(cfgFile, e.cfg); err != nil {
					return err
				}
			}
			// --conn flag overrides config file value.
			if cmd.Flags().Changed("conn") {
				e.cfg.Conn, _ = cmd.Flags().GetString("conn")
			}
			return nil
		},
	}

	root.PersistentFlags().String("conn", "", "PostgreSQL connection string")
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", cfgFile, "Config file path [$BBL_CONFIG]")

	root.AddCommand(newMigrateCmd(e.cfg))
	root.AddCommand(newUsersCmd(e))
	root.AddCommand(newOrganizationsCmd(e))
	root.AddCommand(newPeopleCmd(e))
	root.AddCommand(newProjectsCmd(e))
	root.AddCommand(newWorksCmd(e))
	root.AddCommand(newMutateCmd(e))
	root.AddCommand(newReindexCmd(e))
	root.AddCommand(newSeedCmd(e))
	root.AddCommand(newStartCmd(e))

	return root
}
