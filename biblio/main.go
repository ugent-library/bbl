package main

import (
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/biblio/plato"
	"github.com/ugent-library/bbl/cmd/bbl/cli"
)

func main() {
	bbl.RegisterWorkSource("plato", &plato.WorkSource{})

	cli.Run()
}
