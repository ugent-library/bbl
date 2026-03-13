package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUsersCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
	}
	cmd.AddCommand(newUsersImportSourceCmd(e))
	return cmd
}

func newUsersImportSourceCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "import-source <source>",
		Short: "Import users from a configured source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			src, ok := svc.UserSources[name]
			if !ok {
				return fmt.Errorf("unknown user source %q", name)
			}
			seq, err := src.Iter(ctx)
			if err != nil {
				return err
			}
			n, err := svc.Repo.ImportUsers(ctx, name, e.cfg.UserSources[name].AuthProvider, seq)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: imported %d %s\n", name, n, plural(n, "user", "users"))
			return err
		},
	}
}
