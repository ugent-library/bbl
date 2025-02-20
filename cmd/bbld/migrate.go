package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgadapter"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:       "migrate [up|down]",
	Short:     "Run database migrations",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"up", "down"},
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()
		repo := bbl.NewRepo(pgadapter.New(conn))

		if args[0] == "up" {
			return repo.MigrateUp(cmd.Context())
		} else {
			return repo.MigrateDown(cmd.Context())
		}
	},
}
