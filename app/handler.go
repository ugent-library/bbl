package app

import (
	"net/http"
)

// wrap builds an http.Handler: extract context, call handler, handle errors.
// Context is just data. Error handling is a plain function, not a method on T.
func wrap[T any](
	getCtx func(*http.Request) (T, error),
	onErr func(http.ResponseWriter, *http.Request, error),
	h func(http.ResponseWriter, *http.Request, T) error,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := getCtx(r)
		if err != nil {
			onErr(w, r, err)
			return
		}
		if err = h(w, r, c); err != nil {
			onErr(w, r, err)
		}
	})
}

// group binds a middleware chain, context factory, and error handler.
// Routes become one-liners.
type group[T any] struct {
	chain  chain
	getCtx func(*http.Request) (T, error)
	onErr  func(http.ResponseWriter, *http.Request, error)
}

func newGroup[T any](c chain, getCtx func(*http.Request) (T, error), onErr func(http.ResponseWriter, *http.Request, error)) group[T] {
	return group[T]{chain: c, getCtx: getCtx, onErr: onErr}
}

func (g group[T]) handle(h func(http.ResponseWriter, *http.Request, T) error) http.Handler {
	return g.chain.then(wrap(g.getCtx, g.onErr, h))
}
