package app

import (
	"net/http"

	personViews "github.com/ugent-library/bbl/app/views/backoffice/people"
)

func (app *App) backofficePeople(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	opts, err := bindSearchOpts(r, nil)
	if err != nil {
		return err
	}

	hits, err := app.index.People().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return personViews.Search(c.viewCtx(), hits).Render(r.Context(), w)
}
