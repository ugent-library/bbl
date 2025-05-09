package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	sloghttp "github.com/samber/slog-http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/s3store"
	"github.com/ugent-library/bbl/ctx"
	"github.com/ugent-library/bbl/oaipmh"
	"github.com/ugent-library/bbl/oaiservice"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/oidc"
)

//go:embed static
var staticFS embed.FS

type Config struct {
	Env              string
	BaseURL          string
	Logger           *slog.Logger
	Repo             *pgxrepo.Repo
	Index            bbl.Index
	Store            *s3store.Store
	Secret           []byte
	HashSecret       []byte
	AuthIssuerURL    string
	AuthClientID     string
	AuthClientSecret string
}

func New(config *Config) (http.Handler, error) {
	router := mux.NewRouter()
	router.Use(sloghttp.Recovery)
	router.Use(sloghttp.NewWithConfig(config.Logger.WithGroup("http"), sloghttp.Config{
		WithRequestID: true,
	}))

	// static files
	router.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFS))).Methods("GET")

	// openapi
	// apiServer, err := openapi.NewServer(openapi.NewService(
	// 	config.Repo,
	// 	config.Queue,
	// 	config.Index,
	// ))
	// if err != nil {
	// 	return nil, err
	// }
	// router.HandleFunc("/api/v1/openapi.yaml", openapi.SpecHandler).Methods("GET")
	// router.HandleFunc("/api/v1/docs", openapi.DocsHandler("/api/v1/openapi.yaml")).Methods("GET")
	// router.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", apiServer))

	// oai provider
	oaiProvider, err := oaipmh.NewProvider(oaipmh.ProviderConfig{
		RepositoryName: "Ghent University Institutional Archive",
		BaseURL:        "http://localhost:3000/oai",
		AdminEmails:    []string{"nicolas.steenlant@ugent.be"},
		DeletedRecord:  "persistent",
		Granularity:    "YYYY-MM-DDThh:mm:ssZ",
		// StyleSheet:     "/oai.xsl",
		Backend: oaiservice.New(config.Repo),
		ErrorHandler: func(err error) { // TODO
			config.Logger.Error("oai error", "error", err)
		},
	})
	if err != nil {
		return nil, err
	}

	router.Handle("/oai", oaiProvider).Methods("GET")

	// ui
	assets, err := parseManifest()
	if err != nil {
		return nil, err
	}

	// appCtx := ctx.New(BindAppCtx(router, cookies, assets, config.Env == "development", config.Repo.GetUser))
	appCtx := ctx.New(BindAppCtx(config, router, assets))
	loggedInCtx := appCtx.With(RequireUser)

	router.Handle("/", appCtx.Bind(HomeHandler)).Methods("GET").Name("home")

	authProvider, err := oidc.NewAuth(context.TODO(), oidc.Config{
		IssuerURL:        config.AuthIssuerURL,
		ClientID:         config.AuthClientID,
		ClientSecret:     config.AuthClientSecret,
		RedirectURL:      config.BaseURL + "/auth/callback",
		CookieInsecure:   config.Env == "development",
		CookiePrefix:     "bbl.oidc.",
		CookieHashSecret: config.HashSecret,
		CookieSecret:     config.Secret,
	})
	if err != nil {
		return nil, err
	}

	NewAuthHandler(config.Repo, authProvider).AddRoutes(router, appCtx)
	NewOrganizationHandler(config.Repo, config.Index).AddRoutes(router, loggedInCtx)
	NewPersonHandler(config.Repo, config.Index).AddRoutes(router, loggedInCtx)
	NewProjectHandler(config.Repo, config.Index).AddRoutes(router, loggedInCtx)
	NewWorkHandler(config.Repo, config.Index).AddRoutes(router, loggedInCtx)
	NewFileHandler(config.Store).AddRoutes(router, loggedInCtx)

	return router, nil
}

func parseManifest() (map[string]string, error) {
	manifest, err := staticFS.ReadFile("static/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("couldn't read asset manifest: %w", err)
	}
	assets := make(map[string]string)
	if err := json.Unmarshal(manifest, &assets); err != nil {
		return nil, fmt.Errorf("couldn't parse asset manifest: %w", err)
	}

	return assets, nil
}
