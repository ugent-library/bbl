package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	sloghttp "github.com/samber/slog-http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/backoffice"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/discovery"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/oaipmh"
	"github.com/ugent-library/bbl/oaiservice"
)

//go:embed static
var staticFS embed.FS

func New(config *ctx.Config) (http.Handler, error) {
	csrfProtection := http.NewCrossOriginProtection()

	router := mux.NewRouter()
	router.Use(sloghttp.Recovery)
	router.Use(sloghttp.NewWithConfig(config.Logger.WithGroup("http"), sloghttp.Config{
		WithRequestID: true,
	}))
	router.Use(csrfProtection.Handler)

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

	binder := ctx.Binder(config, router, assets)

	b := bind.New(binder)

	b.OnBindError(func(w http.ResponseWriter, r *http.Request, err error) {
		config.Logger.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})
	b.OnError(func(w http.ResponseWriter, r *http.Request, c *ctx.Ctx, err error) {
		config.Logger.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	})

	err = backoffice.AddRoutes(router, binder, b, config)
	if err != nil {
		return nil, err
	}

	err = discovery.AddRoutes(router, b, config)
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

type chain []func(http.Handler) http.Handler

func (c chain) thenFunc(h http.HandlerFunc) http.Handler {
	return c.then(h)
}

func (c chain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

type handlerCtx interface {
	HandleError(http.ResponseWriter, *http.Request, error)
}

func with[T handlerCtx](getCtx func(*http.Request) (T, error), h func(http.ResponseWriter, *http.Request, T) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := getCtx(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err = h(w, r, c); err != nil {
			c.HandleError(w, r, err)
		}
	})
}

type ctxKey string

func (c ctxKey) String() string {
	return string(c)
}

func getCtx[T handlerCtx](r *http.Request, key ctxKey) (T, error) {
	v := r.Context().Value(key)
	c, ok := v.(T)
	if !ok {
		var t T
		return t, fmt.Errorf("getCtx %s: expected %T but got %T", key, t, v)
	}
	return c, nil
}

func setCtx[T handlerCtx](key ctxKey, newCtx func(r *http.Request) (T, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := newCtx(r) // TODO handle error
			r = r.WithContext(context.WithValue(r.Context(), key, c))
			next.ServeHTTP(w, r)
		})
	}
}

const appCtxKey = "appCtx"

func getAppCtx(r *http.Request) (*appCtx, error) {
	return getCtx[*appCtx](r, appCtxKey)
}

type appCtx struct {
	User *bbl.User
}

func (c *appCtx) HandleError(w http.ResponseWriter, r *http.Request, err error) {
}

type App struct {
	logger  *slog.Logger
	handler http.Handler
}

func NewApp(
	logger *slog.Logger,
) (*App, error) {
	mux := http.NewServeMux()

	baseChain := chain{
		sloghttp.Recovery,
		sloghttp.NewWithConfig(logger.WithGroup("http"), sloghttp.Config{
			WithRequestID: true,
		}),
		http.NewCrossOriginProtection().Handler,
	}

	handler := baseChain.then(mux)

	app := &App{
		logger:  logger,
		handler: handler,
	}

	return app, nil
}

// func SearchWorks(mux *http.ServeMux) {
// 	mux.Handle("GET /backoffice/works", with(newSearchWorksCtx, func(w http.ResponseWriter, r *http.Request, c *searchWorksCtx) error {
// 		if err := c.RequireUser(); err != nil {
// 			return err
// 		}
// 		return nil
// 	}))
// }
