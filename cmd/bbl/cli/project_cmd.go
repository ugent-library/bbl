package cli

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
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
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		rec, err := repo.GetProject(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(rec)
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Projects",
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
		logger := newLogger(cmd.OutOrStdout())

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		riverClient, err := newRiverClient(logger, conn, repo, index)
		if err != nil {
			return err
		}

		res, err := riverClient.Insert(cmd.Context(), jobs.ReindexProjects{}, nil)
		if err != nil {
			return err
		}

		if res.UniqueSkippedAsDuplicate {
			logger.Info("projects reindexer is already running")
		} else {
			logger.Info("started projects reindexer", "job", res.Job.ID)
		}

		return nil
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

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}
