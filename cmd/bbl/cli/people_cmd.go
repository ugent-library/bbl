package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(peopleCmd)
	peopleCmd.AddCommand(searchPeopleCmd)
	searchPeopleCmd.Flags().IntVar(&searchArgs.Limit, "limit", 20, "")
	searchPeopleCmd.Flags().StringVarP(&searchArgs.Query, "query", "q", "", "")
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

		hits, err := index.People().Search(cmd.Context(), searchArgs)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
