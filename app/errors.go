package app

import (
	"errors"
	"net/http"

	"github.com/ugent-library/bbl"
)

var errNotAuthenticated = errors.New("not authenticated")

func (app *App) htmlError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errNotAuthenticated) {
		http.Redirect(w, r, "/backoffice/login", http.StatusFound)
		return
	}
	if errors.Is(err, bbl.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	app.log.Error("handler error", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
