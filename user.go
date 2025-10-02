package bbl

import (
	"time"

	"github.com/ugent-library/vo"
)

const (
	AdminRole   = "admin"
	CuratorRole = "curator"
	UserRole    = "user"

	ViewPermission = "view"
	EditPermission = "edit"
)

var UserRoles = []string{
	AdminRole,
	CuratorRole,
	UserRole,
}

var UserPermissions = []string{
	ViewPermission,
	EditPermission,
}

type User struct {
	ID           string    `json:"id,omitempty"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Identifiers  []Code    `json:"identifiers,omitempty"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at,omitzero"`
	UpdatedAt    time.Time `json:"updated_at,omitzero"`
	DeactivateAt time.Time `json:"deactivate_at,omitzero"`
}

type Permission struct {
	UserID string `json:"user_id"`
	Kind   string `json:"kind"`
}

func (rec *User) Validate() error {
	v := vo.New(
		vo.NotBlank("username", rec.Username),
		vo.NotBlank("email", rec.Email),
		vo.NotBlank("name", rec.Name),
		vo.OneOf("role", rec.Role, UserRoles),
	)

	for i, ident := range rec.Identifiers {
		v.In("identifiers").Index(i).Add(
			vo.NotBlank("scheme", ident.Scheme),
			vo.NotBlank("val", ident.Val),
		)
	}

	return v.Validate().ToError()
}
