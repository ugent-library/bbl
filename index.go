package bbl

import (
	"context"
)

type Index interface {
	Organizations() RecIndex[*Organization]
	People() RecIndex[*Person]
}

type RecIndex[T Rec] interface {
	Add(context.Context, T) error
	Search(context.Context, SearchArgs) (*RecHits[T], error)
	// IndexSwitcher(context.Context) (IndexSwitcher[T], error)
}

type SearchArgs struct {
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

// type IndexSwitcher[T any] interface {
// 	Add(context.Context, T) error
// 	Switch(context.Context) error
// }
