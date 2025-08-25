package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	sloghttp "github.com/samber/slog-http"

	"github.com/ugent-library/bbl/app/backoffice"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/oaipmh"
	"github.com/ugent-library/bbl/oaiservice"
)

//go:embed static
var staticFS embed.FS

func New(config *ctx.Config) (http.Handler, error) {
	router := mux.NewRouter()
	router.Use(sloghttp.Recovery)
	router.Use(sloghttp.NewWithConfig(config.Logger.WithGroup("http"), sloghttp.Config{
		WithRequestID: true,
	}))

	// static files
	router.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFS))).Methods("GET")

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

	b := bind.New(ctx.Binder(config, router, assets))

	err = backoffice.AddRoutes(router, b, config)
	if err != nil {
		return nil, err
	}

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
