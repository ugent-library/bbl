package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var ErrNotFound = errors.New("not found")

type DbAdapter interface {
	MigrateUp(context.Context) error
	MigrateDown(context.Context) error
	GetRecWithKind(context.Context, string, string) (*RawRecord, error)
	Do(context.Context, func(DbTx) error) error
}

type DbTx interface {
	GetRec(context.Context, string) (*RawRecord, error)
	AddRev(context.Context, *Rev) error
}

type RawRecord struct {
	ID    string    `json:"id"`
	Kind  string    `json:"kind"`
	Attrs []*DbAttr `json:"attrs"`
}

// TODO just use work profile
func (r *RawRecord) BaseKind() string {
	baseKind, _, _ := strings.Cut(r.Kind, ".")
	return baseKind
}

type DbAttr struct {
	ID    string          `json:"id"`
	Kind  string          `json:"kind"`
	Val   json.RawMessage `json:"val"`
	RelID string          `json:"rel_id"`
	Rel   *RawRecord      `json:"rel"`
}
