package cli

import (
	"encoding/json"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/workflows"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(searchProjectsCmd)
	searchProjectsCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchProjectsCmd.Flags().IntVar(&searchOpts.Size, "size", 20, "")
	searchProjectsCmd.Flags().IntVar(&searchOpts.From, "from", 0, "")
	searchProjectsCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	projectsCmd.AddCommand(reindexProjectsCmd)
}

var projectCmd = &cobra.Command{
	Use:   "project [id]",
	Short: "Get project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetProject(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return writeData(cmd, rec)
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Projects",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.ProjectsIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var reindexProjectsCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex projects job",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ReindexProjects(hatchetClient, repo, index)

		res, err := task.Run(cmd.Context(), workflows.ReindexProjectsInput{})
		if err != nil {
			return err
		}

		out := workflows.ReindexProjectsOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
	},
}

var searchProjectsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Projects().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
