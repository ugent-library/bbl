package bbl

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

// newID generates a time-ordered ULID stored as a UUID-compatible [16]byte.
// Monotonic ordering reduces B-tree fragmentation on sequential inserts.
func newID() ulid.ULID {
	return ulid.MustNew(ulid.Now(), rand.Reader)
}
