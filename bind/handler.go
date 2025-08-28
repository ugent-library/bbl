package bind

import (
	"context"
	"net/http"
	"slices"
)

type contextKey string

func (c contextKey) String() string {
	return "bind: " + string(c)
}

var ctxKey = contextKey("ctx")

type Handler[T any] interface {
	ServeHTTP(http.ResponseWriter, *http.Request, T) error
}

type HandlerFunc[T any] func(http.ResponseWriter, *http.Request, T) error

func (h HandlerFunc[T]) ServeHTTP(w http.ResponseWriter, r *http.Request, ctx T) error {
	return h(w, r, ctx)
}

type Binder[T any] struct {
	httpMiddleware   []func(http.Handler) http.Handler
	middleware       []func(Handler[T]) Handler[T]
	bindErrorHandler func(http.ResponseWriter, *http.Request, error)
	errorHandler     func(http.ResponseWriter, *http.Request, T, error)
}

func New[T any](binder func(*http.Request) (T, error)) *Binder[T] {
	b := &Binder[T]{
		bindErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		},
		errorHandler: func(w http.ResponseWriter, _ *http.Request, _ T, _ error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
	}
	b.httpMiddleware = []func(http.Handler) http.Handler{
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, err := binder(r)
				if err != nil {
					b.bindErrorHandler(w, r, err)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), ctxKey, ctx))
				next.ServeHTTP(w, r)
			})
		},
	}
	return b
}

func Derive[T, TT any](b *Binder[T], deriver func(*http.Request, T) (TT, error)) *Binder[TT] {
	bb := New(func(r *http.Request) (TT, error) {
		return deriver(r, r.Context().Value(ctxKey).(T))
	})

	mw := func(next http.Handler) http.Handler {
		var h Handler[T] = HandlerFunc[T](func(w http.ResponseWriter, r *http.Request, _ T) error {
			next.ServeHTTP(w, r)
			return nil
		})

		for _, mw := range slices.Backward(b.middleware) {
			h = mw(h)
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context().Value(ctxKey).(T)
			if err := h.ServeHTTP(w, r, ctx); err != nil {
				b.errorHandler(w, r, ctx, err)
			}
		})
	}

	bb.httpMiddleware = append(b.httpMiddleware, mw, bb.httpMiddleware[0])

	return bb
}

func (b *Binder[T]) OnBindError(h func(http.ResponseWriter, *http.Request, error)) {
	b.bindErrorHandler = h
}

func (b *Binder[T]) OnError(h func(http.ResponseWriter, *http.Request, T, error)) {
	b.errorHandler = h
}

func (b *Binder[T]) With(chain ...func(Handler[T]) Handler[T]) *Binder[T] {
	bb := *b
	bb.middleware = append(bb.middleware, chain...)
	return &bb
}

func (b *Binder[T]) Bind(h Handler[T]) http.Handler {
	for _, mw := range slices.Backward(b.middleware) {
		h = mw(h)
	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value(ctxKey).(T)
		if err := h.ServeHTTP(w, r, ctx); err != nil {
			b.errorHandler(w, r, ctx, err)
		}
	})

	for _, mw := range slices.Backward(b.httpMiddleware) {
		handler = mw(handler)
	}

	return handler
}

func (b *Binder[T]) BindFunc(h HandlerFunc[T]) http.Handler {
	return b.Bind(h)
}
