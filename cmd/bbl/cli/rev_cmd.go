package cli

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
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
		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

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
