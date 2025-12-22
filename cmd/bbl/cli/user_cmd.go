package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
)

func init() {
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(importUserSourceCmd)
	usersCmd.AddCommand(searchUsersCmd)
	searchUsersCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchUsersCmd.Flags().IntVar(&searchOpts.Size, "size", 20, "")
	searchUsersCmd.Flags().IntVar(&searchOpts.From, "from", 0, "")
	searchUsersCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	usersCmd.AddCommand(reindexUsersCmd)

}

var userCmd = &cobra.Command{
	Use:   "user [id]",
	Short: "Get user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetUser(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return writeData(cmd, rec)
	},
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Users",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		enc := json.NewEncoder(cmd.OutOrStdout())

		for rec := range repo.UsersIter(cmd.Context(), &err) {
			if err = enc.Encode(rec); err != nil {
				return err
			}
		}

		return err
	},
}

var importUserSourceCmd = &cobra.Command{
	Use:   "import-source",
	Short: "import users from source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		if bbl.GetUserSource(source) == nil {
			return fmt.Errorf("unknown source %s", source)
		}

		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		info, err := repo.Catbird.RunTaskWait(cmd.Context(),
			tasks.ImportUserSourceName,
			tasks.ImportUserSourceInput{Source: source},
			catbird.RunTaskOpts{},
		)
		if err != nil {
			return err
		}

		return writeData(cmd, info)
	},
}

var reindexUsersCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex users job",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		info, err := repo.Catbird.RunTaskWait(cmd.Context(),
			tasks.ReindexUsersName,
			tasks.ReindexUsersInput{},
			catbird.RunTaskOpts{DeduplicationID: tasks.ReindexUsersName},
		)
		if err != nil {
			return err
		}

		return writeData(cmd, info)
	},
}

var searchUsersCmd = &cobra.Command{
	Use:   "search",
	Short: "Search users",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hits, err := index.Users().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		return writeData(cmd, hits)
	},
}
