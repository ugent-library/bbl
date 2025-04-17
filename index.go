package bbl

import (
	"context"
)

type Index interface {
	Organizations() RecIndex[*Organization]
	People() RecIndex[*Person]
	Projects() RecIndex[*Project]
	Works() RecIndex[*Work]
}

type RecIndex[T Rec] interface {
	Add(context.Context, T) error
	Search(context.Context, SearchOpts) (*RecHits[T], error)
	NewSwitcher(context.Context) (RecIndexSwitcher[T], error)
}

type RecIndexSwitcher[T Rec] interface {
	Add(context.Context, T) error
	Switch(context.Context) error
}

type SearchOpts struct {
	Cursor string `json:"cursor,omitempty"`
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit"`
}

type RecHits[T Rec] struct {
	Hits   []RecHit[T] `json:"hits"`
	Total  int         `json:"total"`
	Cursor string      `json:"cursor,omitempty"`
	Query  string      `json:"query,omitempty"`
	Limit  int         `json:"limit"`
}

type RecHit[T any] struct {
	Rec T `json:"rec"`
}
