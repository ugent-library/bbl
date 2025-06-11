package bbl

import "time"

const (
	AdminRole   = "admin"
	CuratorRole = "curator"
	UserRole    = "user"

	ViewPermission = "view"
	EditPermission = "edit"
)

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Identifiers []Code    `json:"identifiers,omitempty"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Permission struct {
	UserID string `json:"user_id"`
	Kind   string `json:"kind"`
}
