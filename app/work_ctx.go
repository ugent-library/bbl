package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ctx"
)

type WorkCtx struct {
	*AppCtx
	Work *bbl.Work
}

func BindWorkCtx(repo *bbl.Repo) ctx.Deriver[*AppCtx, *WorkCtx] {
	return func(r *http.Request, appCtx *AppCtx) (*WorkCtx, error) {
		work, err := repo.GetWork(r.Context(), mux.Vars(r)["work_id"])
		if err != nil {
			return nil, err
		}
		return &WorkCtx{AppCtx: appCtx, Work: work}, nil
	}
}
