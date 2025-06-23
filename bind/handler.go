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

type HandlerBinder[T any] struct {
	binderChain []interface {
		wrap(http.Handler) http.Handler
	}
	binder           func(*http.Request) (T, error)
	middlewareChain  []func(Handler[T]) Handler[T]
	bindErrorHandler func(http.ResponseWriter, *http.Request, error)
	errorHandler     func(http.ResponseWriter, *http.Request, T, error)
}

func New[T any](binder func(*http.Request) (T, error)) *HandlerBinder[T] {
	return &HandlerBinder[T]{
		binder: binder,
		bindErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		},
		errorHandler: func(w http.ResponseWriter, _ *http.Request, _ T, _ error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
	}
}

func (b *HandlerBinder[T]) OnBindError(h func(http.ResponseWriter, *http.Request, error)) {
	b.bindErrorHandler = h
}

func (b *HandlerBinder[T]) OnError(h func(http.ResponseWriter, *http.Request, T, error)) {
	b.errorHandler = h
}

func (b *HandlerBinder[T]) With(chain ...func(Handler[T]) Handler[T]) *HandlerBinder[T] {
	bb := *b
	bb.middlewareChain = append(bb.middlewareChain, chain...)
	return &bb
}

func (b *HandlerBinder[T]) wrap(h http.Handler) http.Handler {
	return b.Bind(HandlerFunc[T](func(w http.ResponseWriter, r *http.Request, _ T) error {
		h.ServeHTTP(w, r)
		return nil
	}))
}

func (b *HandlerBinder[T]) Bind(h Handler[T]) (httpHandler http.Handler) {
	for _, mw := range slices.Backward(b.middlewareChain) {
		h = mw(h)
	}

	httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := b.binder(r)
		if err != nil {
			b.bindErrorHandler(w, r, err)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxKey, ctx))
		if err := h.ServeHTTP(w, r, ctx); err != nil {
			b.errorHandler(w, r, ctx, err)
			return
		}
	})

	for _, binder := range slices.Backward(b.binderChain) {
		httpHandler = binder.wrap(httpHandler)
	}

	return
}

func (b *HandlerBinder[T]) BindFunc(h HandlerFunc[T]) (httpHandler http.Handler) {
	return b.Bind(h)
}

func Derive[T, TT any](b *HandlerBinder[T], deriver func(*http.Request, T) (TT, error)) *HandlerBinder[TT] {
	return &HandlerBinder[TT]{
		binderChain: append(b.binderChain, b),
		binder: func(r *http.Request) (TT, error) {
			t := r.Context().Value(ctxKey).(T)
			return deriver(r, t)
		},
	}
}
