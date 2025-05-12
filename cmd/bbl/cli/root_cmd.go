package cli

import (
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

var rootCmd = &cobra.Command{
	Use: "bbl",
}

var searchOpts = &bbl.SearchOpts{}
