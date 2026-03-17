package app

import (
	"net/http"
	"slices"
)

// chain is a composable list of middleware.
type chain []func(http.Handler) http.Handler

// with returns a new chain with additional middleware appended.
// func (c chain) with(mw ...func(http.Handler) http.Handler) chain {
// 	return append(c, mw...)
// }

// then applies the chain to a handler, outermost first.
func (c chain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}
