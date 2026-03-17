package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newMutateCmd(e *env) *cobra.Command {
	var userIDFlag string

	cmd := &cobra.Command{
		Use:   "mutate",
		Short: "Apply mutations from JSONL stdin",
		Long: `Read mutation objects from stdin (one JSON object per line) and apply them
as a single revision.

Example:
  echo '{"mutation": "set_work_volume", "work_id": "01J...", "val": "42"}' | bbl mutate --user 01J...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}

			var userID *bbl.ID
			if userIDFlag != "" {
				id, err := bbl.ParseID(userIDFlag)
				if err != nil {
					return fmt.Errorf("invalid user ID: %w", err)
				}
				userID = &id
			}

			var mutations []any
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					continue
				}
				m, err := bbl.DecodeMutation(line)
				if err != nil {
					return err
				}
				mutations = append(mutations, m)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}

			if len(mutations) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no mutations to apply")
				return nil
			}

			ok, _, err := svc.Repo.Mutate(ctx, userID, mutations...)
			if err != nil {
				return err
			}
			if ok {
				fmt.Fprintf(cmd.OutOrStdout(), "applied %d %s\n", len(mutations), plural(len(mutations), "mutation", "mutations"))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes (all mutations were noops)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user", "", "user ID")

	return cmd
}
