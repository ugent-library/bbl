package bbl

import (
	"context"
	"encoding/json"
)

type DbAdapter interface {
	MigrateUp(context.Context) error
	MigrateDown(context.Context) error
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
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Seq  int
	Val json.RawMessage `json:"val"`
}
