package bbl

import (
	"context"
	"iter"
)

// UserSource is implemented by packages that stream user records from an
// external directory (LDAP, SCIM, CSV, …). Iter connects eagerly and returns
// a fatal setup error (e.g. connection refused, bad credentials) as the second
// return value. Per-entry errors are yielded inline so the caller can skip
// individual bad records without aborting the sweep.
type UserSource interface {
	Iter(ctx context.Context) (iter.Seq2[*ImportUserInput, error], error)
}
