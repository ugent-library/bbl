package app

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type AuthProvider interface {
	BeginAuth(http.ResponseWriter, *http.Request) error
	CompleteAuth(http.ResponseWriter, *http.Request, any) error
}

type AuthHandler struct {
	repo     *pgxrepo.Repo
	provider AuthProvider
}

func NewAuthHandler(repo *pgxrepo.Repo, provider AuthProvider) *AuthHandler {
	return &AuthHandler{
		repo:     repo,
		provider: provider,
	}
}

func (h *AuthHandler) AddRoutes(r *mux.Router, b *bind.HandlerBinder[*AppCtx]) {
	r.Handle("/login", b.BindFunc(h.Login)).Methods("GET").Name("login")
	r.Handle("/auth/callback", b.BindFunc(h.AuthCallback)).Methods("GET")
	r.Handle("/logout", b.BindFunc(h.Logout)).Methods("GET").Name("logout")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	return h.provider.BeginAuth(w, r)
}

func (h *AuthHandler) AuthCallback(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var claims json.RawMessage

	if err := h.provider.CompleteAuth(w, r, &claims); err != nil {
		return err
	}

	username := gjson.GetBytes(claims, "sub").String() // TODO make username claim configurable

	user, err := h.repo.GetUser(r.Context(), "username:"+username)
	if err != nil {
		return err
	}

	if err = c.SetUser(w, user); err != nil {
		return err
	}

	http.Redirect(w, r, c.Route("home").String(), http.StatusFound)

	return nil
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	c.ClearUser(w)
	http.Redirect(w, r, c.Route("home").String(), http.StatusFound)
	return nil
}
