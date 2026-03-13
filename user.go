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
	ID            ID
	CreatedAt     time.Time
	Username      string
	Email         string
	Name          string
	Role          string
	DeactivateAt  *time.Time
	PersonID      *ID
	AuthProviders []AuthProvider
}

type UserAttrs struct {
	Username string
	Email    string
	Name     string
	Role     string
}

// UserIdentifier is an auth claim identifier (e.g. scheme="ugent_id", val="abc123").
type UserIdentifier struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
}

// ImportUserInput carries all data for one user record arriving from a source.
// Role is only applied on creation; subsequent imports do not overwrite a role
// set by an admin.
type ImportUserInput struct {
	Source       string           `json:"source,omitempty"`
	SourceID    string           `json:"source_id"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"` // nil = permanent; set for recurring directory sources
	Username    string           `json:"username"`
	Email       string           `json:"email"`
	Name        string           `json:"name"`
	Role        string           `json:"role,omitempty"`
	Identifiers []UserIdentifier `json:"identifiers,omitempty"`
	AuthProvider string          `json:"auth_provider,omitempty"` // optional — name of the auth provider this source drives (e.g. "ugent_oidc")
}
