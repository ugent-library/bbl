package app

import (
	"net/http"

	workviews "github.com/ugent-library/bbl/app/views/discovery/works"
)

func (app *App) works(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	opts, err := bindSearchOpts(r, []string{"kind"})
	if err != nil {
		return err
	}
	opts.AddTermsFilter("status", "public")

	hits, err := app.index.Works().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return workviews.Search(c.viewCtx(), hits).Render(r.Context(), w)
}

func (app *App) work(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := app.index.Works().Get(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	return workviews.Show(c.viewCtx(), rec).Render(r.Context(), w)
}
