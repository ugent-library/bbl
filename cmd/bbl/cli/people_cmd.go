package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newPeopleCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "people",
		Short: "Manage people",
	}
	cmd.AddCommand(newPeopleImportCmd(e))
	cmd.AddCommand(newPeopleGetCmd(e))
	cmd.AddCommand(newPeopleListCmd(e))
	cmd.AddCommand(newPeopleSearchCmd(e))
	cmd.AddCommand(newPeopleSearchAllCmd(e))
	return cmd
}

func newPeopleImportCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "import <source>",
		Short: "Import people from stdin (JSONL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			source := args[0]
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			if err := svc.Repo.UpsertSource(ctx, source); err != nil {
				return err
			}
			seq := func(yield func(*bbl.ImportPersonInput, error) bool) {
				dec := json.NewDecoder(os.Stdin)
				for {
					var raw json.RawMessage
					if err := dec.Decode(&raw); err == io.EOF {
						return
					} else if err != nil {
						yield(nil, err)
						return
					}
					var v bbl.ImportPersonInput
					if err := json.Unmarshal(raw, &v); err != nil {
						yield(nil, err)
						return
					}
					v.SourceRecord = raw
					if !yield(&v, nil) {
						return
					}
				}
			}
			n, err := svc.Repo.ImportPeople(ctx, source, seq)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: imported %d %s\n", source, n, plural(n, "person", "people"))
			return err
		},
	}
}

func newPeopleGetCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a person by ID (JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			id, err := bbl.ParseID(args[0])
			if err != nil {
				return fmt.Errorf("invalid ID: %w", err)
			}
			person, err := svc.Repo.GetPerson(ctx, id)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), person)
		},
	}
}

func newPeopleListCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all people (JSONL)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			for person, err := range svc.Repo.EachPerson(ctx) {
				if err != nil {
					return err
				}
				if err := writeJSON(cmd.OutOrStdout(), person); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newPeopleSearchCmd(e *env) *cobra.Command {
	var q, filter string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search people (JSONL)",
		Long:  "Search people and return results as JSONL.\n\n" + filterHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			if svc.Index == nil {
				return fmt.Errorf("no search index configured")
			}
			opts := &bbl.SearchOpts{
				Query: q,
				Size:  limit,
			}
			if filter != "" {
				f, err := bbl.ParseQueryFilter(filter)
				if err != nil {
					return fmt.Errorf("invalid filter: %w", err)
				}
				opts.Filter = f
			}
			hits, err := svc.Index.People().Search(ctx, opts)
			if err != nil {
				return err
			}
			for _, h := range hits.Hits {
				if err := writeJSON(cmd.OutOrStdout(), h); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&q, "query", "q", "", "search query (omit for match_all)")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"status=public\")")
	cmd.Flags().IntVar(&limit, "limit", 100, "max results to return")
	return cmd
}

func newPeopleSearchAllCmd(e *env) *cobra.Command {
	var q, filter string
	cmd := &cobra.Command{
		Use:   "search-all",
		Short: "Search all people, cursor-tailing (JSONL)",
		Long:  "Search all people using cursor-based pagination and return results as JSONL.\n\n" + filterHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			if svc.Index == nil {
				return fmt.Errorf("no search index configured")
			}
			opts := &bbl.SearchOpts{Query: q}
			if filter != "" {
				f, err := bbl.ParseQueryFilter(filter)
				if err != nil {
					return fmt.Errorf("invalid filter: %w", err)
				}
				opts.Filter = f
			}
			for h, err := range bbl.SearchAllPeople(ctx, svc.Index.People(), opts) {
				if err != nil {
					return err
				}
				if err := writeJSON(cmd.OutOrStdout(), h); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&q, "query", "q", "", "search query (omit for match_all)")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"status=public\")")
	return cmd
}
