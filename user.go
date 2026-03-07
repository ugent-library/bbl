package bbl

import "time"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// AuthProvider is an entry in User.AuthProviders.
// Stored as a jsonb array to allow additional fields (e.g. added_at) in future.
type AuthProvider struct {
	Provider string `json:"provider"`
}

type User struct {
	ID               string
	CreatedAt        time.Time
	Username         string
	Email            string
	Name             string
	Role             string
	DeactivateAt     *time.Time
	PersonIdentityID *string
	AuthProviders    []AuthProvider
}

type UserAttrs struct {
	Username string
	Email    string
	Name     string
	Role     string
}

// UserIdentifier is an auth claim identifier (e.g. scheme="ugent_id", val="abc123").
type UserIdentifier struct {
	Scheme string
	Val    string
}

// ImportUserInput carries all data for one user record arriving from a source.
// Role is only applied on creation; subsequent imports do not overwrite a role
// set by an admin.
type ImportUserInput struct {
	Source         string
	SourceRecordID string
	ExpiresAt      *time.Time // nil = permanent; set for recurring directory sources
	Username       string
	Email          string
	Name           string
	Role           string
	Identifiers    []UserIdentifier
	AuthProvider   string // optional — name of the auth provider this source drives (e.g. "ugent_oidc")
}
