package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/cmd/bbl/cli"
	"github.com/ugent-library/bbl/ugent/platosource"
)

func main() {
	reg := &cli.Registry{}
	cli.RegisterWorkSourceIter(reg, "plato", platosource.New)
	cobra.CheckErr(cli.NewRootCmd(reg).ExecuteContext(context.Background()))
}
