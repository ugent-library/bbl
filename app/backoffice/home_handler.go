package backoffice

import (
	"net/http"

	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views/backoffice"
)

func HomeHandler(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	return backoffice.Home(c.ViewCtx()).Render(r.Context(), w)
}
