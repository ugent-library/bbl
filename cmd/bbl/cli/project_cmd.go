package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
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

		info, err := repo.Catbird.RunTaskWait(cmd.Context(),
			tasks.ReindexProjectsName,
			tasks.ReindexProjectsInput{},
			catbird.RunTaskOpts{DeduplicationID: tasks.ReindexProjectsName},
		)
		if err != nil {
			return err
		}

		return writeData(cmd, info)
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

		return writeData(cmd, hits)
	},
}
