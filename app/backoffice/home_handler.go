package backoffice

import (
	"net/http"

	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views"
)

func HomeHandler(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	return views.Home(c.ViewCtx()).Render(r.Context(), w)
}
