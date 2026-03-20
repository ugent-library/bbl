package bbl

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("conflict")
	ErrCuratorLock = errors.New("field is locked by a curator")
)
