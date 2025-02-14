package main

import (
	"encoding/json"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugentlib/bbl"
	"github.com/ugentlib/bbl/pgadapter"
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
		repo := bbl.NewRepo(pgadapter.New(conn))

		var changes []*bbl.Change

		dec := json.NewDecoder(cmd.InOrStdin())
		for {
			var c bbl.Change
			if err := dec.Decode(&c); err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			changes = append(changes, &c)
		}

		if err := repo.AddRev(cmd.Context(), changes); err != nil {
			return err
		}

		return nil
	},
}
