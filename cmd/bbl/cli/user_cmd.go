package cli

import (
	"encoding/json"
	"fmt"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/workflows"
)

func init() {
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(importUserSourceCmd)
}

var userCmd = &cobra.Command{
	Use:   "user [id]",
	Short: "Get user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

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
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

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

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ImportUserSource(hatchetClient, repo)

		res, err := task.Run(cmd.Context(), workflows.ImportUserSourceInput{Source: source})
		if err != nil {
			return err
		}

		out := workflows.ImportUserSourceOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
	},
}
