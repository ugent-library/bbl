package ctx

import (
	"context"
	"log"
	"net/http"
)

// TODO make implementation less clunky; is closure based possible?
// TODO pass ctx value as an argument instead of through Context
type contextKey string

func (c contextKey) String() string {
	return string(c)
}

var ctxKey = contextKey("ctx")

type Binder[T any] func(*http.Request) (T, error)

type Deriver[T, TT any] func(*http.Request, T) (TT, error)

// return values control middleware execution: halt if returned request is nil otherwise
// continue to the next inner wrapper; an error always halts the execution flow
type Wrapper[T any] func(http.ResponseWriter, *http.Request, T) (*http.Request, error)

type Handler[T any] func(http.ResponseWriter, *http.Request, T) error

type Ctx[T any] struct {
	chain []interface {
		applyWrappers(http.ResponseWriter, *http.Request) (*http.Request, error)
	}
	binder   Binder[T]
	wrappers []Wrapper[T]
}

func New[T any](binder Binder[T], wrappers ...Wrapper[T]) *Ctx[T] {
	return &Ctx[T]{
		binder:   binder,
		wrappers: wrappers,
	}
}

func Derive[T, TT any](c *Ctx[T], deriver Deriver[T, TT], wrappers ...Wrapper[TT]) *Ctx[TT] {
	return &Ctx[TT]{
		chain: append(c.chain, c),
		binder: func(r *http.Request) (TT, error) {
			t := r.Context().Value(ctxKey).(T)
			return deriver(r, t)
		},
		wrappers: wrappers,
	}
}

func (c *Ctx[T]) With(wrappers ...Wrapper[T]) *Ctx[T] {
	return &Ctx[T]{
		chain:    c.chain,
		binder:   c.binder,
		wrappers: append(c.wrappers, wrappers...),
	}
}

func (c *Ctx[T]) applyWrappers(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	t, err := c.binder(r)
	if err != nil {
		return nil, err
	}
	req := r
	for i := len(c.wrappers) - 1; i >= 0; i-- {
		req, err = c.wrappers[i](w, r, t)
		if err != nil {
			return nil, err
		}
		if req == nil {
			return nil, nil
		}
	}

	return req.WithContext(context.WithValue(req.Context(), ctxKey, t)), nil
}

func (c *Ctx[T]) Bind(h Handler[T]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r
		var err error
		for _, chained := range append(c.chain, c) {
			req, err = chained.applyWrappers(w, req)
			if err != nil {
				// TODO error handler
				log.Printf("error: %s", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req == nil {
				return
			}
		}

		t := req.Context().Value(ctxKey).(T)
		if err := h(w, r, t); err != nil {
			// TODO error handler
			log.Printf("error: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
