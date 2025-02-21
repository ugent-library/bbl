package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	sloghttp "github.com/samber/slog-http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ctx"
	// h "github.com/ugent-library/minibiblio/app/handlers"
)

//go:embed static
var staticFiles embed.FS

type Config struct {
	Env     string
	BaseURL string
	Logger  *slog.Logger
	Repo    *bbl.Repo
	// Queue            biblio.Queue
	// Index            biblio.Index
	// UserSource       biblio.UserSource
	CookieSecret     []byte
	CookieHashSecret []byte
	AuthIssuerURL    string
	AuthClientID     string
	AuthClientSecret string
}

func New(config *Config) (http.Handler, error) {
	cookies := securecookie.New(config.CookieHashSecret, config.CookieSecret)
	cookies.SetSerializer(securecookie.JSONEncoder{})

	router := mux.NewRouter()
	router.Use(sloghttp.Recovery)
	router.Use(sloghttp.NewWithConfig(config.Logger.WithGroup("http"), sloghttp.Config{
		WithRequestID: true,
	}))

	// static files
	router.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFiles))).Methods("GET")

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
	// oaiProvider, err := oaipmh.NewProvider(oaipmh.ProviderConfig{
	// 	RepositoryName: "Ghent University Institutional Archive",
	// 	BaseURL:        "http://localhost:3000/oai",
	// 	AdminEmails:    []string{"nicolas.steenlant@ugent.be"},
	// 	DeletedRecord:  "persistent",
	// 	Granularity:    "YYYY-MM-DDThh:mm:ssZ",
	// 	// StyleSheet:     "/oai.xsl",
	// 	Backend: oai.NewService(config.Repo.WorkRepresentations()),
	// 	ErrorHandler: func(err error) { // TODO
	// 		config.Logger.Error("oai error", "error", err)
	// 	},
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// router.Handle("/oai", oaiProvider).Methods("GET")

	// ui
	assets, err := parseManifest()
	if err != nil {
		return nil, err
	}

	// appCtx := ctx.New(h.BindAppCtx(router, cookies, assets, config.Env == "development", config.Repo.Users()))
	appCtx := ctx.New(BindAppCtx(router, cookies, assets, config.Env == "development"))
	// loggedInCtx := appCtx.With(h.RequireUser)

	router.Handle("/", appCtx.Bind(HomeHandler)).Methods("GET").Name("home")

	// authProvider, err := oidc.NewAuth(context.TODO(), oidc.Config{
	// 	IssuerURL:        config.AuthIssuerURL,
	// 	ClientID:         config.AuthClientID,
	// 	ClientSecret:     config.AuthClientSecret,
	// 	RedirectURL:      config.BaseURL + "/auth/callback",
	// 	CookieInsecure:   config.Env == "development",
	// 	CookiePrefix:     "biblio.oidc.",
	// 	CookieHashSecret: config.CookieHashSecret,
	// 	CookieSecret:     config.CookieSecret,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// h.NewAuthHandler(h.AuthConfig{
	// 	Provider:   authProvider,
	// 	UserSource: config.UserSource,
	// 	UsersRepo:  config.Repo.Users(),
	// }).AddRoutes(router, appCtx)

	// h.NewWorkHandler(config.Repo).AddRoutes(router, loggedInCtx)
	NewWorkHandler(config.Repo).AddRoutes(router, appCtx)

	// h.NewSearchWorksHandler(config.Repo.Works(), config.Index).AddRoutes(router, loggedInCtx)

	return router, nil
}

func parseManifest() (map[string]string, error) {
	manifest, err := staticFiles.ReadFile("static/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("couldn't read asset manifest: %w", err)
	}
	assets := make(map[string]string)
	if err := json.Unmarshal(manifest, &assets); err != nil {
		return nil, fmt.Errorf("couldn't parse asset manifest: %w", err)
	}

	return assets, nil
}
