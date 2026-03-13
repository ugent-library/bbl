package cli

import (
	"fmt"
	"iter"
	"time"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newReindexCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reindex <works|people|projects|organizations>",
		Short: "Rebuild a search index from the database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			if svc.Index == nil {
				return fmt.Errorf("no search index configured (set opensearch in config)")
			}

			start := time.Now()
			entity := args[0]

			switch entity {
			case "works":
				err = svc.Index.Works().Reindex(ctx,
					svc.Repo.EachWork(ctx),
					func(since time.Time) iter.Seq2[*bbl.Work, error] {
						return svc.Repo.EachWorkSince(ctx, since)
					},
				)
			case "people":
				err = svc.Index.People().Reindex(ctx,
					svc.Repo.EachPerson(ctx),
					func(since time.Time) iter.Seq2[*bbl.Person, error] {
						return svc.Repo.EachPersonSince(ctx, since)
					},
				)
			case "projects":
				err = svc.Index.Projects().Reindex(ctx,
					svc.Repo.EachProject(ctx),
					func(since time.Time) iter.Seq2[*bbl.Project, error] {
						return svc.Repo.EachProjectSince(ctx, since)
					},
				)
			case "organizations":
				err = svc.Index.Organizations().Reindex(ctx,
					svc.Repo.EachOrganization(ctx),
					func(since time.Time) iter.Seq2[*bbl.Organization, error] {
						return svc.Repo.EachOrganizationSince(ctx, since)
					},
				)
			default:
				return fmt.Errorf("unknown entity type %q; expected works, people, projects, or organizations", entity)
			}

			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "reindex %s completed in %s\n", entity, time.Since(start).Round(time.Millisecond))
			return nil
		},
	}
	return cmd
}
