package app

import (
	"net/http"

	"github.com/ugent-library/bbl/app/views"
)

func HomeHandler(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	return views.Home(c.ViewCtx()).Render(r.Context(), w)
}
