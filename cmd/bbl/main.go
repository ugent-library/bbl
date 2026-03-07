package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/cmd/bbl/cli"
)

func main() {
	cobra.CheckErr(cli.NewRootCmd(cli.Registry{}).ExecuteContext(context.Background()))
}
