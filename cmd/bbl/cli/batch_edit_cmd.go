package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newWorksBatchExportCmd(e *env) *cobra.Command {
	var ids, filter string

	cmd := &cobra.Command{
		Use:   "batch-export",
		Short: "Export works as batch-edit CSV",
		Long: `Export scalar fields for a set of works as a CSV suitable for batch editing.

Examples:
  bbl works batch-export --ids 01JXYZ,01JABC > edit.csv
  bbl works batch-export --filter "status=public kind=journal_article" > edit.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}

			var workIDs []bbl.ID
			switch {
			case ids != "":
				for _, s := range strings.Split(ids, ",") {
					id, err := bbl.ParseID(strings.TrimSpace(s))
					if err != nil {
						return fmt.Errorf("invalid ID %q: %w", s, err)
					}
					workIDs = append(workIDs, id)
				}
			case filter != "":
				if svc.Index == nil {
					return fmt.Errorf("no search index configured")
				}
				f, err := bbl.ParseQueryFilter(filter)
				if err != nil {
					return fmt.Errorf("invalid filter: %w", err)
				}
				for work, err := range svc.SearchAllWorkRecords(ctx, &bbl.SearchOpts{Filter: f}) {
					if err != nil {
						return err
					}
					workIDs = append(workIDs, work.ID)
				}
			default:
				return fmt.Errorf("specify --ids or --filter")
			}

			if len(workIDs) == 0 {
				return fmt.Errorf("no works found")
			}

			return bbl.WriteWorkBatch(ctx, svc.Repo, cmd.OutOrStdout(), workIDs)
		},
	}

	cmd.Flags().StringVar(&ids, "ids", "", "comma-separated work IDs")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression")

	return cmd
}

func newWorksBatchImportCmd(e *env) *cobra.Command {
	var userIDFlag string

	cmd := &cobra.Command{
		Use:   "batch-import",
		Short: "Apply batch-edit CSV from stdin",
		Long: `Read a batch-edit CSV from stdin (as exported by batch-export),
diff against current values, and apply changes.

Example:
  cat edit.csv | bbl works batch-import --user 01J...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}

			if userIDFlag == "" {
				return fmt.Errorf("--user is required")
			}
			userID, err := bbl.ParseID(userIDFlag)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}

			result, err := bbl.ReadWorkBatch(ctx, svc.Repo, os.Stdin)
			if err != nil {
				return err
			}

			if len(result.Conflicts) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: %d field(s) changed since export:\n", len(result.Conflicts))
				for _, c := range result.Conflicts {
					fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s: current=%q csv=%q\n",
						c.WorkID, c.Field, c.CurrentVal, c.CSVVal)
				}
			}

			updates := result.Updates()
			if len(updates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes to apply")
				return nil
			}

			ok, _, err := svc.Repo.Update(ctx, &userID, updates...)
			if err != nil {
				return err
			}
			if ok {
				fmt.Fprintf(cmd.OutOrStdout(), "applied %d %s\n",
					len(updates), plural(len(updates), "change", "changes"))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes (all updates were noops)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user", "", "user ID (required)")

	return cmd
}
