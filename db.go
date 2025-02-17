package bbl

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrNotFound = errors.New("not found")

type DbAdapter interface {
	MigrateUp(context.Context) error
	MigrateDown(context.Context) error
	GetRecWithKind(context.Context, string, string) (*DbRec, error)
	Do(context.Context, func(DbTx) error) error
}

type DbTx interface {
	GetRec(context.Context, string) (*DbRec, error)
	AddRev(context.Context, *Rev) error
}

type DbRec struct {
	ID    string    `json:"id"`
	Kind  string    `json:"kind"`
	Attrs []*DbAttr `json:"attrs"`
}

type DbAttr struct {
	ID    string          `json:"id"`
	Kind  string          `json:"kind"`
	Val   json.RawMessage `json:"val"`
	RelID string          `json:"rel_id"`
	Rel   json.RawMessage `json:"rel"`
}
