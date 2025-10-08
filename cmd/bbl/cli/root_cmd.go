package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"github.com/ugent-library/bbl"
)

var (
	prettify    = false
	searchOpts  = &bbl.SearchOpts{}
	queryFilter = ""
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&prettify, "pretty", false, "")
}

var rootCmd = &cobra.Command{
	Use: "bbl",
}

func writeData(cmd *cobra.Command, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return writeJSON(cmd, b)
}

func writeJSON(cmd *cobra.Command, b []byte) error {
	if prettify {
		b = pretty.Color(pretty.Pretty(b), pretty.TerminalStyle)
	}
	_, err := cmd.OutOrStdout().Write(b)
	return err
}
