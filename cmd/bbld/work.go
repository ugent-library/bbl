package main

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugentlib/bbl"
	"github.com/ugentlib/bbl/pgadapter"
)

func init() {
	rootCmd.AddCommand(workCmd)
}

var workCmd = &cobra.Command{
	Use:   "work [id]",
	Short: "Get work",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()
		repo := bbl.NewRepo(pgadapter.New(conn))

		rec, err := repo.GetWork(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(rec)
	},
}
