package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func init() {
	rootCmd.AddCommand(peopleCmd)
	peopleCmd.AddCommand(searchPeopleCmd)
}

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "People",
}

var searchPeopleCmd = &cobra.Command{
	Use:   "search",
	Short: "Search people",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.People().Search(cmd.Context(), bbl.SearchArgs{Limit: 20})
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
