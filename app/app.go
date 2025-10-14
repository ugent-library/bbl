package app

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/securecookie"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/leonelquinteros/gotext"
	sloghttp "github.com/samber/slog-http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/urls"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/i18n"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/s3store"
	"github.com/ugent-library/crypt"
	"github.com/ugent-library/oidc"
)

const (
	sessionCookieName = "bbl.session"
)

//go:embed static
var staticFS embed.FS

type SessionCookie struct {
	UserID string `json:"u"`
}

type AuthProvider interface {
	BeginAuth(http.ResponseWriter, *http.Request) error
	CompleteAuth(http.ResponseWriter, *http.Request, any) error
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

func (c chain) with(mw ...func(http.Handler) http.Handler) chain {
	return append(c, mw...)
}

// func (c chain) thenFunc(h http.HandlerFunc) http.Handler {
// 	return c.then(h)
// }

func (c chain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

type handlerCtx interface {
	HandleError(http.ResponseWriter, *http.Request, error)
}

func wrap[T handlerCtx](getCtx func(*http.Request) (T, error), h func(http.ResponseWriter, *http.Request, T) error) http.Handler {
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

const appCtxKey ctxKey = "appCtx"

func getAppCtx(r *http.Request) (*appCtx, error) {
	return getCtx[*appCtx](r, appCtxKey)
}

type appCtx struct {
	insecure bool
	assets   map[string]string
	crypt    *crypt.Crypt
	cookies  *securecookie.SecureCookie
	User     *bbl.User
	Hub      *catbird.Hub
	topics   []string
	Loc      *gotext.Locale
}

func (c *appCtx) viewCtx() views.Ctx {
	return views.Ctx{
		AssetPath: c.AssetPath,
		SSEPath:   c.SSEPath,
		Loc:       c.Loc,
		User:      c.User,
	}
}

func (c *appCtx) HandleError(w http.ResponseWriter, r *http.Request, err error) {
}

func (c *appCtx) AddTopic(topic string) {
	if !slices.Contains(c.topics, topic) {
		c.topics = append(c.topics, topic)
	}
}

func (c *appCtx) SSEPath() string {
	token, err := c.crypt.EncryptValue(c.topics)
	if err != nil {
		panic(err)
	}
	return urls.BackofficeSSE(token)
}

func (c *appCtx) AssetPath(asset string) string {
	a, ok := c.assets[asset]
	if !ok {
		panic(fmt.Errorf("asset '%s' not found in manifest", asset))
	}
	return a
}

func (c *appCtx) SetUser(w http.ResponseWriter, user *bbl.User) error {
	val := &SessionCookie{
		UserID: user.ID,
	}
	err := c.SetCookie(w, sessionCookieName, val, 30*24*time.Hour)
	if err != nil {
		return err
	}

	c.User = user

	return nil
}

func (c *appCtx) ClearUser(w http.ResponseWriter) {
	c.User = nil
	c.ClearCookie(w, sessionCookieName)
}

func (c *appCtx) GetCookie(r *http.Request, name string, val any) error {
	cookie, err := r.Cookie(name)
	if err != nil {
		return fmt.Errorf("can't get cookie %s: %w", name, err)
	}
	if err := c.cookies.Decode(name, cookie.Value, val); err != nil {
		return fmt.Errorf("can't decode cookie %s: %w", name, err)
	}
	return nil
}

func (c *appCtx) SetCookie(w http.ResponseWriter, name string, val any, ttl time.Duration) error {
	v, err := c.cookies.Encode(name, val)
	if err != nil {
		return fmt.Errorf("can't encode cookie %s: %w", name, err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    v,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   !c.insecure,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

func (c *appCtx) ClearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !c.insecure,
		SameSite: http.SameSiteStrictMode,
	})
}

type App struct {
	env             string
	logger          *slog.Logger
	repo            *pgxrepo.Repo
	store           *s3store.Store
	index           bbl.Index
	crypt           *crypt.Crypt
	assets          map[string]string
	cookies         *securecookie.SecureCookie
	hub             *catbird.Hub
	authProvider    AuthProvider
	exportWorksTask *hatchet.StandaloneTask
}

func NewApp(
	baseURL string,
	env string,
	logger *slog.Logger,
	hashSecret, secret []byte,
	repo *pgxrepo.Repo,
	store *s3store.Store,
	index bbl.Index,
	hub *catbird.Hub,
	authIssuerURL string,
	authClientID string,
	authClientSecret string,
	exportWorksTask *hatchet.StandaloneTask,
) (*App, error) {
	assets, err := parseManifest()
	if err != nil {
		return nil, err
	}

	cookies := securecookie.New(hashSecret, secret)
	cookies.SetSerializer(securecookie.JSONEncoder{})

	authProvider, err := oidc.NewAuth(context.TODO(), oidc.Config{
		IssuerURL:        authIssuerURL,
		ClientID:         authClientID,
		ClientSecret:     authClientSecret,
		RedirectURL:      baseURL + "/backoffice/auth/callback",
		CookieInsecure:   env == "development",
		CookiePrefix:     "bbl.oidc.",
		CookieHashSecret: hashSecret,
		CookieSecret:     secret,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		env:             env,
		logger:          logger,
		repo:            repo,
		store:           store,
		index:           index,
		crypt:           crypt.New(secret),
		assets:          assets,
		cookies:         cookies,
		hub:             hub,
		authProvider:    authProvider,
		exportWorksTask: exportWorksTask,
	}

	return app, nil
}

func (app *App) Handler() http.Handler {
	baseChain := chain{
		sloghttp.Recovery,
		sloghttp.NewWithConfig(app.logger.WithGroup("http"), sloghttp.Config{
			WithRequestID: true,
		}),
		http.NewCrossOriginProtection().Handler,
	}

	appChain := chain{setCtx(appCtxKey, app.newAppCtx)}

	userChain := appChain.with(requireUser)

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	mux.Handle("GET /works/{id}", appChain.then(wrap(getAppCtx, app.work)))
	mux.Handle("GET /works", appChain.then(wrap(getAppCtx, app.works)))

	mux.Handle("GET /backoffice/login", appChain.then(wrap(getAppCtx, app.login)))
	mux.Handle("GET /backoffice/auth/callback", appChain.then(wrap(getAppCtx, app.authCallback)))
	mux.Handle("GET /backoffice/logout", appChain.then(wrap(getAppCtx, app.logout)))

	mux.Handle("GET /backoffice/organizations", userChain.then(wrap(getAppCtx, app.backofficeOrganizations)))

	mux.Handle("GET /backoffice/people", userChain.then(wrap(getAppCtx, app.backofficePeople)))

	mux.Handle("GET /backoffice/projects", userChain.then(wrap(getAppCtx, app.backofficeProjects)))

	mux.Handle("GET /backoffice/works/add", userChain.then(wrap(getAppCtx, app.backofficeAddWork)))
	mux.Handle("GET /backoffice/works/new", userChain.then(wrap(getAppCtx, app.backofficeNewWork)))
	mux.Handle("POST /backoffice/works/export/{format}", userChain.then(wrap(getAppCtx, app.backofficeExportWorks)))
	mux.Handle("GET /backoffice/works/batch_edit", userChain.then(wrap(getAppCtx, app.backofficeBatchEditWorks)))
	mux.Handle("POST /backoffice/works/batch_edit", userChain.then(wrap(getAppCtx, app.backofficeBatchUpdateWorks)))
	mux.Handle("POST /backoffice/works/_change_kind", userChain.then(wrap(getAppCtx, app.backofficeWorkChangeKind)))
	mux.Handle("POST /backoffice/works/_add_identifier", userChain.then(wrap(getAppCtx, app.backofficeWorkAddIdentifier)))
	mux.Handle("POST /backoffice/works/_remove_identifier", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveIdentifier)))
	mux.Handle("GET /backoffice/works/_suggest_contributor", userChain.then(wrap(getAppCtx, app.backofficeWorkSuggestContributor)))
	mux.Handle("POST /backoffice/works/_add_contributor", userChain.then(wrap(getAppCtx, app.backofficeWorkAddContributor)))
	mux.Handle("GET /backoffice/works/_edit_contributor", userChain.then(wrap(getAppCtx, app.backofficeWorkEditContributor)))
	mux.Handle("POST /backoffice/works/_remove_contributor", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveContributor)))
	mux.Handle("POST /backoffice/works/_add_files", userChain.then(wrap(getAppCtx, app.backofficeWorkAddFiles)))
	mux.Handle("POST /backoffice/works/_remove_file", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveFile)))
	mux.Handle("POST /backoffice/works/_add_title", userChain.then(wrap(getAppCtx, app.backofficeWorkAddTitle)))
	mux.Handle("POST /backoffice/works/_remove_title", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveTitle)))
	mux.Handle("POST /backoffice/works/_add_abstract", userChain.then(wrap(getAppCtx, app.backofficeWorkAddAbstract)))
	mux.Handle("GET /backoffice/works/_edit_abstract", userChain.then(wrap(getAppCtx, app.backofficeWorkEditAbstract)))
	mux.Handle("POST /backoffice/works/_remove_lay_summary", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveLaySummary)))
	mux.Handle("POST /backoffice/works/_add_lay_summary", userChain.then(wrap(getAppCtx, app.backofficeWorkAddLaySummary)))
	mux.Handle("GET /backoffice/works/_edit_lay_summary", userChain.then(wrap(getAppCtx, app.backofficeWorkEditLaySummary)))
	mux.Handle("POST /backoffice/works/_remove_abstract", userChain.then(wrap(getAppCtx, app.backofficeWorkRemoveAbstract)))
	mux.Handle("GET /backoffice/works", userChain.then(wrap(getAppCtx, app.backofficeWorks)))
	mux.Handle("POST /backoffice/works", userChain.then(wrap(getAppCtx, app.backofficeCreateWork)))
	mux.Handle("GET /backoffice/works/{id}/changes", userChain.then(wrap(getAppCtx, app.backofficeWorkChanges)))
	mux.Handle("GET /backoffice/works/{id}/edit", userChain.then(wrap(getAppCtx, app.backofficeEditWork)))
	mux.Handle("POST /backoffice/works/{id}", userChain.then(wrap(getAppCtx, app.backofficeUpdateWork)))

	mux.Handle("POST /backoffice/files/upload_url", userChain.then(wrap(getAppCtx, app.createFileUploadURL)))

	mux.Handle("GET /backoffice/sse", userChain.then(wrap(getAppCtx, app.backofficeSSE)))

	mux.Handle("GET /backoffice", userChain.then(wrap(getAppCtx, app.backofficeHome)))

	return baseChain.then(mux)
}

func (app *App) newAppCtx(r *http.Request) (*appCtx, error) {
	c := &appCtx{
		insecure: app.env == "development",
		assets:   app.assets,
		crypt:    app.crypt,
		cookies:  app.cookies,
		Loc:      i18n.Locales["en"],
		Hub:      app.hub,
	}

	// get user from session if present
	session := SessionCookie{}
	err := c.GetCookie(r, sessionCookieName, &session)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return nil, err
	}
	if err == nil {
		user, err := app.repo.GetUser(r.Context(), session.UserID)
		if err != nil {
			return nil, err
		}
		c.User = user
		c.AddTopic("users")
		c.AddTopic("users." + user.ID)
	}

	return c, nil
}

func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := getAppCtx(r)
		if err != nil { // TODO log error
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if c.User == nil {
			http.Redirect(w, r, urls.BackofficeLogin(), http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
