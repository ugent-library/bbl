package cli

import (
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/queryparser"
)

var workFormat string
var worksFormat string

func init() {
	rootCmd.AddCommand(workCmd)
	workCmd.Flags().StringVar(&workFormat, "format", "json", "")
	rootCmd.AddCommand(worksCmd)
	worksCmd.Flags().StringVar(&worksFormat, "format", "jsonl", "")
	worksCmd.AddCommand(searchWorksCmd)
	searchWorksCmd.Flags().StringVarP(&searchOpts.Query, "query", "q", "", "")
	searchWorksCmd.Flags().StringVarP(&queryFilter, "filter", "f", "", "")
	searchWorksCmd.Flags().IntVar(&searchOpts.Size, "size", 20, "")
	searchWorksCmd.Flags().IntVar(&searchOpts.From, "from", 0, "")
	searchWorksCmd.Flags().StringVar(&searchOpts.Cursor, "cursor", "", "")
	worksCmd.AddCommand(reindexWorksCmd)
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

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		rec, err := repo.GetWork(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		b, err := bbl.EncodeWork(rec, workFormat)
		if err != nil {
			return err
		}

		_, err = cmd.OutOrStdout().Write(b)

		return err
	},
}

var worksCmd = &cobra.Command{
	Use:   "works",
	Short: "Works",
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

		e, err := bbl.NewWorkExporter(cmd.OutOrStdout(), worksFormat)
		if err != nil {
			return err
		}

		for rec := range repo.WorksIter(cmd.Context(), &err) {
			if err := e.Add(rec); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}

		return e.Done()
	},
}

var reindexWorksCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex works job",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := newLogger(cmd.OutOrStdout())

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		riverClient, err := newInsertOnlyRiverClient(logger, conn)
		if err != nil {
			return err
		}

		res, err := riverClient.Insert(cmd.Context(), jobs.ReindexWorks{}, nil)
		if err != nil {
			return err
		}

		if res.UniqueSkippedAsDuplicate {
			logger.Info("works reindexer is already running")
		} else {
			logger.Info("started works reindexer", "job", res.Job.ID)
		}

		return reportJobProgress(cmd.Context(), riverClient, res.Job.ID, logger)
	},
}

var searchWorksCmd = &cobra.Command{
	Use:   "search",
	Short: "Search works",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		// TODO organize this
		if queryFilter != "" {
			ast, err := queryparser.ParseReader("filter", strings.NewReader(queryFilter))
			if err != nil {
				return err
			}
			searchOpts.Filter = andClauseFrom(ast)
		}

		hits, err := index.Works().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hits)
	},
}

func andClauseFrom(node any) *bbl.AndClause {
	var f bbl.Filter
	switch n := node.(type) {
	case *queryparser.AndQuery:
		f = filterFromAndQuery(n)
	case *queryparser.Query:
		f = filterFromQuery(n)
	case *queryparser.Field:
		f = filterFromField(n)
	}
	if andClause, ok := f.(*bbl.AndClause); ok {
		return andClause
	} else {
		return bbl.And(f)
	}
}

func filterFromAndQuery(n *queryparser.AndQuery) bbl.Filter {
	f := &bbl.AndClause{Filters: make([]bbl.Filter, len(n.FieldOrQueries))}
	for i, fq := range n.FieldOrQueries {
		if fq.Field != nil {
			f.Filters[i] = filterFromField(fq.Field)
		} else {
			f.Filters[i] = filterFromQuery(fq.Query)
		}
	}
	if len(f.Filters) == 1 {
		return f.Filters[0]
	}
	return f
}

func filterFromQuery(n *queryparser.Query) bbl.Filter {
	if len(n.OrQueries) > 0 {
		f := &bbl.OrClause{Filters: make([]bbl.Filter, 1+len(n.OrQueries))}
		f.Filters[0] = filterFromAndQuery(n.AndQuery)
		for i, aq := range n.OrQueries {
			f.Filters[i+1] = filterFromAndQuery(aq)
		}
		return f
	}
	return filterFromAndQuery(n.AndQuery)
}

func filterFromField(n *queryparser.Field) bbl.Filter {
	return bbl.Terms(n.Key.Name, n.Value.(string))
}
