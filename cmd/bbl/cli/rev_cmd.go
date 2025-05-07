package cli

import (
	"encoding/json"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

func init() {
	rootCmd.AddCommand(revCmd)
	revCmd.AddCommand(addRevCmd)
}

var revCmd = &cobra.Command{
	Use:   "rev",
	Short: "Revisions",
}

var addRevCmd = &cobra.Command{
	Use:   "add",
	Short: "Add revision",
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

		dec := json.NewDecoder(cmd.InOrStdin())
		for {
			rev := &bbl.Rev{}
			if err := dec.Decode(rev); err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			if err := repo.AddRev(cmd.Context(), rev); err != nil {
				return err
			}
		}

		return nil
	},
}
