package discovery

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
)

func AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx], config *ctx.Config) error {
	NewWorksHandler(config.Index).AddRoutes(r, b)

	return nil
}
