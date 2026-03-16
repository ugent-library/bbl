package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	sloghttp "github.com/samber/slog-http"
	"github.com/ugent-library/bbl"
)

type Config struct {
	Logger   *slog.Logger
	Services *bbl.Services
	RootURL  string // public root URL (e.g. "https://biblio.ugent.be/bbl")
	Dev      bool   // dev mode: serve assets from disk, re-read manifest on every request

	// Auth — all nil-safe. Without auth, session is not created,
	// User is always nil, and login/callback return 404.
	// Keyed by provider name (e.g. "ugent_oidc").
	Auth       map[string]AuthProvider
	HashSecret []byte // HMAC key for session cookie signing
	Secret     []byte // encryption key for session cookie
	Secure     bool   // true = HTTPS-only cookies
}

type App struct {
	log        *slog.Logger
	services   *bbl.Services
	rootURL    string // without trailing slash
	pathPrefix string // path component of rootURL (e.g. "/bbl" or "")
	assets     *assets
	locale     *locale
	auth       map[string]AuthProvider
	session    *session // nil when no auth configured
}

func New(cfg Config) (*App, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	rootURL := strings.TrimRight(cfg.RootURL, "/")
	var pathPrefix string
	if rootURL != "" {
		if u, err := url.Parse(rootURL); err == nil {
			pathPrefix = strings.TrimRight(u.Path, "/")
		}
	}

	a, err := loadAssets(pathPrefix, cfg.Dev)
	if err != nil {
		return nil, fmt.Errorf("load assets: %w", err)
	}
	loc, err := loadLocale(cfg.Dev)
	if err != nil {
		return nil, fmt.Errorf("load locale: %w", err)
	}
	app := &App{
		log:        log,
		services:   cfg.Services,
		rootURL:    rootURL,
		pathPrefix: pathPrefix,
		assets:     a,
		locale:     loc,
		auth:       cfg.Auth,
	}
	if len(cfg.Auth) > 0 {
		app.session = newSession(cfg.HashSecret, cfg.Secret, cfg.Secure)
	}
	return app, nil
}

func (app *App) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Static assets (CSS, JS) with immutable cache headers.
	mux.Handle("GET /static/", http.StripPrefix("/static/", app.assets.fileServer()))

	// Language switcher — sets cookie and redirects back.
	mux.HandleFunc("GET /lang/{lang}", app.switchLang)

	// SRU endpoints — per-entity, public.
	mux.Handle("GET /sru/works", app.sruWorksHandler())

	base := chain{
		sloghttp.Recovery,
		sloghttp.NewWithConfig(app.log.WithGroup("http"), sloghttp.Config{
			WithRequestID: true,
		}),
	}

	// Discovery — public, anonymous. User loaded from session if present.
	discovery := newGroup(base, app.newCtx, app.htmlError)
	mux.Handle("GET /", discovery.handle(app.home))
	mux.Handle("GET /works", discovery.handle(app.searchWorks))
	mux.Handle("GET /works/{id}", discovery.handle(app.showWork))
	mux.Handle("GET /people", discovery.handle(app.searchPeople))
	mux.Handle("GET /people/{id}", discovery.handle(app.showPerson))
	mux.Handle("GET /projects", discovery.handle(app.searchProjects))
	mux.Handle("GET /projects/{id}", discovery.handle(app.showProject))
	mux.Handle("GET /organizations", discovery.handle(app.searchOrganizations))
	mux.Handle("GET /organizations/{id}", discovery.handle(app.showOrganization))

	// Auth routes — anonymous (login/callback don't require a session).
	mux.Handle("GET /backoffice/login", discovery.handle(app.login))
	mux.Handle("GET /backoffice/login/{provider}", discovery.handle(app.loginProvider))
	mux.Handle("GET /backoffice/auth/callback/{provider}", discovery.handle(app.authCallback))

	// Backoffice — requires authenticated user.
	backoffice := newGroup(base, app.newAuthCtx, app.htmlError)
	mux.Handle("GET /backoffice", backoffice.handle(app.backofficeHome))
	mux.Handle("GET /backoffice/works", backoffice.handle(app.backofficeSearchWorks))
	mux.Handle("GET /backoffice/works/{id}", backoffice.handle(app.backofficeShowWork))
	mux.Handle("GET /backoffice/works/{id}/edit", backoffice.handle(app.backofficeEditWork))
	mux.Handle("POST /backoffice/works/{id}/edit", backoffice.handle(app.backofficeUpdateWork))
	mux.Handle("GET /backoffice/people", backoffice.handle(app.backofficeSearchPeople))
	mux.Handle("GET /backoffice/people/{id}", backoffice.handle(app.backofficeShowPerson))
	mux.Handle("GET /backoffice/projects", backoffice.handle(app.backofficeSearchProjects))
	mux.Handle("GET /backoffice/projects/{id}", backoffice.handle(app.backofficeShowProject))
	mux.Handle("GET /backoffice/organizations", backoffice.handle(app.backofficeSearchOrganizations))
	mux.Handle("GET /backoffice/organizations/{id}", backoffice.handle(app.backofficeShowOrganization))
	mux.Handle("POST /backoffice/logout", backoffice.handle(app.logout))

	return mux
}
