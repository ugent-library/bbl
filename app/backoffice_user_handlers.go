package app

import (
	"net/http"

	userviews "github.com/ugent-library/bbl/app/views/backoffice/users"
)

func (app *App) backofficeUsers(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	opts, err := bindSearchOpts(r, nil)
	if err != nil {
		return err
	}

	hits, err := app.index.Users().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return userviews.Search(c.viewCtx(), hits).Render(r.Context(), w)
}
