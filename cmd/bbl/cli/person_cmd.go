package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
)

func init() {
	rootCmd.AddCommand(personCmd)
	rootCmd.AddCommand(peopleCmd)
	peopleCmd.AddCommand(searchPeopleCmd)
	searchPeopleCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchPeopleCmd.Flags().IntVar(&searchOpts.Size, "size", 20, "")
	searchPeopleCmd.Flags().IntVar(&searchOpts.From, "from", 0, "")
	searchPeopleCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	peopleCmd.AddCommand(reindexPeopleCmd)
}

var personCmd = &cobra.Command{
	Use:   "person [id]",
	Short: "Get person",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetPerson(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return writeData(cmd, rec)
	},
}

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "People",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.PeopleIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var reindexPeopleCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex people job",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		info, err := repo.Catbird.RunTaskWait(cmd.Context(),
			tasks.ReindexPeopleName,
			tasks.ReindexPeopleInput{},
			catbird.RunTaskOpts{DeduplicationID: tasks.ReindexPeopleName},
		)
		if err != nil {
			return err
		}

		return writeData(cmd, info)
	},
}

var searchPeopleCmd = &cobra.Command{
	Use:   "search",
	Short: "Search people",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.People().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		return writeData(cmd, hits)
	},
}
