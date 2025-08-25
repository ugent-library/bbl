package backoffice

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/oidc"
)

func AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx], config *ctx.Config) error {
	r = r.PathPrefix("/backoffice/").Subrouter()

	authProvider, err := oidc.NewAuth(context.TODO(), oidc.Config{
		IssuerURL:        config.AuthIssuerURL,
		ClientID:         config.AuthClientID,
		ClientSecret:     config.AuthClientSecret,
		RedirectURL:      config.BaseURL + "/backoffice/auth/callback",
		CookieInsecure:   config.Env == "development",
		CookiePrefix:     "bbl.oidc.",
		CookieHashSecret: config.HashSecret,
		CookieSecret:     config.Secret,
	})
	if err != nil {
		return err
	}

	requireUser := b.With(ctx.RequireUser)

	r.Handle("/", b.BindFunc(HomeHandler)).Methods("GET").Name("home")
	r.Handle("/sse", requireUser.BindFunc(SSEHandler)).Methods("GET").Name("sse")
	NewAuthHandler(config.Repo, authProvider).AddRoutes(r, b)
	NewOrganizationHandler(config.Repo, config.Index).AddRoutes(r, requireUser)
	NewPersonHandler(config.Repo, config.Index).AddRoutes(r, requireUser)
	NewProjectHandler(config.Repo, config.Index).AddRoutes(r, requireUser)
	NewWorkHandler(config.Repo, config.Index).AddRoutes(r, requireUser)
	NewFileHandler(config.Store).AddRoutes(r, requireUser)

	return nil
}
