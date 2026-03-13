package bbl

import (
	"crypto/rand"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/oklog/ulid/v2"
)

// ID is a sortable, UUID-compatible unique identifier.
// Generated as a ID for sequential B-tree inserts; stored as the PostgreSQL
// uuid type. Implements pgtype.UUIDScanner and pgtype.UUIDValuer so pgx works
// directly with the raw bytes, bypassing string-format conversion.
type ID [16]byte

// ScanUUID implements pgtype.UUIDScanner for pgx v5 uuid column scanning.
func (u *ID) ScanUUID(v pgtype.UUID) error {
	if v.Valid {
		*u = ID(v.Bytes)
	}
	return nil
}

// UUIDValue implements pgtype.UUIDValuer for pgx v5 uuid column encoding.
func (u ID) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: [16]byte(u), Valid: true}, nil
}

// String returns the ID string representation (base32 Crockford).
func (u ID) String() string {
	return ulid.ULID(u).String()
}

// MarshalText implements encoding.TextMarshaler.
func (u ID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
// Accepts both ULID (Crockford base32) and UUID (hyphenated hex) formats,
// since PostgreSQL's uuid type serializes as hyphenated hex in JSON.
func (u *ID) UnmarshalText(b []byte) error {
	s := string(b)

	// Try ULID first (26 chars, Crockford base32).
	if parsed, err := ulid.ParseStrict(s); err == nil {
		*u = ID(parsed)
		return nil
	}

	// Try UUID (36 chars, hyphenated hex).
	var pg pgtype.UUID
	if err := pg.Scan(s); err == nil && pg.Valid {
		*u = ID(pg.Bytes)
		return nil
	}

	return fmt.Errorf("invalid ID %q: expected ULID or UUID format", b)
}

// ParseID parses a string into an ID. Returns an error if the string is invalid.
func ParseID(s string) (ID, error) {
	var id ID
	if err := id.UnmarshalText([]byte(s)); err != nil {
		return ID{}, err
	}
	return id, nil
}

// newID generates a time-ordered ID. Monotonic ordering reduces B-tree
// fragmentation on sequential inserts.
func newID() ID {
	return ID(ulid.MustNew(ulid.Now(), rand.Reader))
}
