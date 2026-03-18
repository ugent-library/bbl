package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newUpdateCmd(e *env) *cobra.Command {
	var userIDFlag string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Apply updates from JSONL stdin",
		Long: `Read update objects from stdin (one JSON object per line) and apply them
as a single revision.

Example:
  echo '{"set": "work_volume", "work_id": "01J...", "val": "42"}' | bbl update --user 01J...`,
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

			var updates []any
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					continue
				}
				m, err := bbl.DecodeUpdate(line)
				if err != nil {
					return err
				}
				updates = append(updates, m)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}

			if len(updates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no updates to apply")
				return nil
			}

			ok, _, err := svc.Repo.Update(ctx, userID, updates...)
			if err != nil {
				return err
			}
			if ok {
				fmt.Fprintf(cmd.OutOrStdout(), "applied %d %s\n", len(updates), plural(len(updates), "update", "updates"))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes (all updates were noops)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user", "", "user ID")

	return cmd
}
