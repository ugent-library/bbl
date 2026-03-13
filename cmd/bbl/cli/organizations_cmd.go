package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

func newOrganizationsCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organizations",
		Short: "Manage organizations",
	}
	cmd.AddCommand(newOrganizationsImportCmd(e))
	cmd.AddCommand(newOrganizationsGetCmd(e))
	cmd.AddCommand(newOrganizationsListCmd(e))
	cmd.AddCommand(newOrganizationsSearchCmd(e))
	cmd.AddCommand(newOrganizationsSearchAllCmd(e))
	return cmd
}

func newOrganizationsImportCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "import <source>",
		Short: "Import organizations from stdin (JSONL)",
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
			seq := func(yield func(*bbl.ImportOrganizationInput, error) bool) {
				dec := json.NewDecoder(os.Stdin)
				for {
					var raw json.RawMessage
					if err := dec.Decode(&raw); err == io.EOF {
						return
					} else if err != nil {
						yield(nil, err)
						return
					}
					var v bbl.ImportOrganizationInput
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
			n, err := svc.Repo.ImportOrganizations(ctx, source, seq)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: imported %d %s\n", source, n, plural(n, "organization", "organizations"))
			return err
		},
	}
}

func newOrganizationsGetCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an organization by ID (JSON)",
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
			org, err := svc.Repo.GetOrganization(ctx, id)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), org)
		},
	}
}

func newOrganizationsListCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organizations (JSONL)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			for org, err := range svc.Repo.EachOrganization(ctx) {
				if err != nil {
					return err
				}
				if err := writeJSON(cmd.OutOrStdout(), org); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newOrganizationsSearchCmd(e *env) *cobra.Command {
	var q, filter string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search organizations (JSONL)",
		Long:  "Search organizations and return results as JSONL.\n\n" + filterHelp,
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
			hits, err := svc.Index.Organizations().Search(ctx, opts)
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
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"kind=faculty\")")
	cmd.Flags().IntVar(&limit, "limit", 100, "max results to return")
	return cmd
}

func newOrganizationsSearchAllCmd(e *env) *cobra.Command {
	var q, filter string
	cmd := &cobra.Command{
		Use:   "search-all",
		Short: "Search all organizations, cursor-tailing (JSONL)",
		Long:  "Search all organizations using cursor-based pagination and return results as JSONL.\n\n" + filterHelp,
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
			for h, err := range bbl.SearchAllOrganizations(ctx, svc.Index.Organizations(), opts) {
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
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "filter expression (e.g. \"kind=faculty\")")
	return cmd
}
