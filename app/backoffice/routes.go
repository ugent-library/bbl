package backoffice

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
)

func AddRoutes(r *mux.Router, binder func(*http.Request) (*ctx.Ctx, error), b *bind.Binder[*ctx.Ctx], config *ctx.Config) error {
	r = r.PathPrefix("/backoffice/").Subrouter()

	requireUser := b.With(ctx.RequireUser)

	NewWorksHandler(binder, config.Repo, config.Index, config.ExportWorksTask).AddRoutes(r, requireUser)

	return nil
}
