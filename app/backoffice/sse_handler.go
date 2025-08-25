package backoffice

import (
	"net/http"

	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/catbird"
)

func SSEHandler(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	var topics []string
	if err := c.DecryptValue(r.URL.Query().Get("token"), &topics); err != nil {
		return err
	}
	return c.Hub.ConnectSSE(w, r, catbird.ConnectOpts{
		Topics: topics,
	})
}
