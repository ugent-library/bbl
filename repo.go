package bbl

import (
	"errors"
	"time"

	"go.breu.io/ulid"
)

var ErrNotFound = errors.New("not found")

func NewID() string {
	return ulid.Make().UUIDString()
}

type GetWorkRepresentationsOpts struct {
	WorkID       string
	Scheme       string
	Limit        int
	UpdatedAtLTE time.Time
	UpdatedAtGTE time.Time
}

// TODO Repo interface
