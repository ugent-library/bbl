package cli

import (
	"fmt"
	"iter"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newWorksCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "works",
		Short: "Manage works",
	}
	cmd.AddCommand(newWorksImportCmd(e))
	cmd.AddCommand(newWorksImportSourceCmd(e))
	cmd.AddCommand(newWorksGetCmd(e))
	cmd.AddCommand(newWorksListCmd(e))
	cmd.AddCommand(newWorksSearchCmd(e))
	cmd.AddCommand(newWorksSearchAllCmd(e))
	return cmd
}

func newWorksImportCmd(e *env) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "import <source>",
		Short: "Import works from stdin",
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
			seq, err := bbl.ReadWorks(os.Stdin, format)
			if err != nil {
				return err
			}
			n, err := svc.Repo.ImportWorks(ctx, source, seq)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: imported %d %s\n", source, n, plural(n, "work", "works"))
			return err
		},
	}
	cmd.Flags().StringVarP(&format, "format", "F", "jsonl", "input format ("+bbl.WorkReaderFormatsHelp()+")")
	return cmd
}

func newWorksGetCmd(e *env) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a work by ID",
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
			work, err := svc.Repo.GetWork(ctx, id)
			if err != nil {
				return err
			}
			if format == "" {
				return writeJSON(cmd.OutOrStdout(), work)
			}
			b, err := bbl.EncodeWork(format, work)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(b)
			return err
		},
	}
	cmd.Flags().StringVarP(&format, "format", "F", "", "output format ("+bbl.WorkEncoderFormatsHelp()+")")
	return cmd
}

func newWorksListCmd(e *env) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all works",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			if format == "" {
				format = "jsonl"
			}
			ww, err := bbl.NewWorkWriter(format)
			if err != nil {
				return err
			}
			_, err = bbl.WriteWorks(cmd.OutOrStdout(), ww, svc.Repo.EachWork(ctx))
			return err
		},
	}
	cmd.Flags().StringVarP(&format, "format", "F", "", "output format ("+bbl.WorkWriterFormatsHelp()+", default: jsonl)")
	return cmd
}

func newWorksSearchCmd(e *env) *cobra.Command {
	var q, filter, format string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search works",
		Long:  "Search works and return results.\n\nWithout -F, returns search hits (JSONL). With -F, fetches full records.\n\n" + filterHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
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
			if format != "" {
				ww, err := bbl.NewWorkWriter(format)
				if err != nil {
					return err
				}
				res, err := svc.SearchWorkRecords(ctx, opts)
				if err != nil {
					return err
				}
				if err := ww.Begin(cmd.OutOrStdout()); err != nil {
					return err
				}
				for _, h := range res.Hits {
					if err := ww.Encode(cmd.OutOrStdout(), h.Work); err != nil {
						return err
					}
				}
				return ww.End(cmd.OutOrStdout())
			}
			if svc.Index == nil {
				return fmt.Errorf("no search index configured")
			}
			hits, err := svc.Index.Works().Search(ctx, opts)
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
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"status=public kind=book|article\")")
	cmd.Flags().StringVarP(&format, "format", "F", "", "output format ("+bbl.WorkWriterFormatsHelp()+"); fetches full records")
	cmd.Flags().IntVar(&limit, "limit", 100, "max results to return")
	return cmd
}

func newWorksSearchAllCmd(e *env) *cobra.Command {
	var q, filter, format string
	cmd := &cobra.Command{
		Use:   "search-all",
		Short: "Search all works, cursor-tailing",
		Long:  "Search all works using cursor-based pagination.\n\nWithout -F, returns search hits (JSONL). With -F, fetches full records.\n\n" + filterHelp,
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
			if format != "" {
				ww, err := bbl.NewWorkWriter(format)
				if err != nil {
					return err
				}
				_, err = bbl.WriteWorks(cmd.OutOrStdout(), ww, svc.SearchAllWorkRecords(ctx, opts))
				return err
			}
			for h, err := range bbl.SearchAllWorks(ctx, svc.Index.Works(), opts) {
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
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"status=public kind=book|article\")")
	cmd.Flags().StringVarP(&format, "format", "F", "", "output format ("+bbl.WorkWriterFormatsHelp()+"); fetches full records")
	return cmd
}

func newWorksImportSourceCmd(e *env) *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "import-source <source>",
		Short: "Import works from a configured source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			source := args[0]
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}

			var seq iter.Seq2[*bbl.ImportWorkInput, error]

			if id != "" {
				src, ok := svc.WorkGetSources[source]
				if !ok {
					return fmt.Errorf("source %q does not support --id", source)
				}
				rec, err := src.Get(ctx, id)
				if err != nil {
					return err
				}
				seq = func(yield func(*bbl.ImportWorkInput, error) bool) {
					yield(rec, nil)
				}
			} else {
				src, ok := svc.WorkIterSources[source]
				if !ok {
					return fmt.Errorf("unknown work source %q", source)
				}
				seq, err = src.Iter(ctx)
				if err != nil {
					return err
				}
			}

			n, err := svc.Repo.ImportWorks(ctx, source, seq)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: imported %d %s\n", source, n, plural(n, "work", "works"))
			return err
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "import a single record by source ID")
	return cmd
}
