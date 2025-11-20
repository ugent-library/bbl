package cli

import (
	"encoding/json"
	"fmt"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/workflows"
	"github.com/ugent-library/vo"
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
	worksCmd.AddCommand(importWorkCmd)
	worksCmd.AddCommand(importWorkSourceCmd)
	worksCmd.AddCommand(importWorkCmd)
	worksCmd.AddCommand(validateWorkCmd)
}

var workCmd = &cobra.Command{
	Use:   "work [id]",
	Short: "Get work",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		rec, err := repo.GetWork(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		b, err := bbl.EncodeWork(rec, workFormat)
		if err != nil {
			return err
		}

		// JSON can be pretty printed
		if args[0] == "json" {
			return writeJSON(cmd, b)
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
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

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

var searchWorksCmd = &cobra.Command{
	Use:   "search",
	Short: "Search works",
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		if queryFilter != "" {
			searchOpts.QueryFilter, err = bbl.ParseQueryFilter(queryFilter)
			if err != nil {
				return err
			}
		}

		hits, err := index.Works().Search(cmd.Context(), searchOpts)
		if err != nil {
			return err
		}

		return writeData(cmd, hits)
	},
}

var importWorkCmd = &cobra.Command{
	Use:   "import",
	Short: "import work from source",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO check importer exists

		source := args[0]
		id := args[1]

		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ImportWork(hatchetClient, repo)

		res, err := task.Run(cmd.Context(), workflows.ImportWorkInput{Source: source, ID: id})
		if err != nil {
			return err
		}

		out := workflows.ImportWorkOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
	},
}

var importWorkSourceCmd = &cobra.Command{
	Use:   "import-source",
	Short: "import works from source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		if bbl.GetWorkSource(source) == nil {
			return fmt.Errorf("unknown source %s", source)
		}

		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ImportWorkSource(hatchetClient, repo)

		res, err := task.Run(cmd.Context(), workflows.ImportWorkSourceInput{Source: source})
		if err != nil {
			return err
		}

		out := workflows.ImportWorkSourceOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
	},
}

var reindexWorksCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Start reindex works job",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		task := workflows.ReindexWorks(hatchetClient, repo, index)

		res, err := task.Run(cmd.Context(), workflows.ReindexWorksInput{})
		if err != nil {
			return err
		}

		out := workflows.ReindexWorksOutput{}
		if err := res.Into(&out); err != nil {
			return err
		}

		return writeData(cmd, out)
	},
}

var validateWorkCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate work",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dec := json.NewDecoder(cmd.InOrStdin())
		var rec bbl.Work
		if err := dec.Decode(&rec); err != nil {
			return err
		}
		if err := bbl.LoadWorkProfile(&rec); err != nil {
			return err
		}
		err := rec.Validate()
		if _, ok := err.(vo.Errors); ok {
			err = writeData(cmd, err)
		}
		return err
	},
}
