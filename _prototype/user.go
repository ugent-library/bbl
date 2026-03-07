package bbl

import (
	"slices"
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
	Header
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	DeactivateAt time.Time `json:"deactivate_at,omitzero"`
}

type Permission struct {
	UserID string `json:"user_id"`
	Kind   string `json:"kind"`
}

func (rec *User) Validate() error {
	v := vo.New(
		vo.NotBlank("username", rec.Username),
		vo.EmailAddress("email", rec.Email),
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

func (rec *User) Diff(rec2 *User) map[string]any {
	changes := map[string]any{}

	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if rec.Username != rec2.Username {
		changes["username"] = rec.Username
	}
	if rec.Email != rec2.Email {
		changes["email"] = rec.Email
	}
	if rec.Name != rec2.Name {
		changes["name"] = rec.Name
	}
	if rec.Role != rec2.Role {
		changes["role"] = rec.Role
	}
	if !rec.DeactivateAt.Equal(rec2.DeactivateAt) {
		if rec.DeactivateAt.IsZero() {
			changes["deactivate_at"] = nil
		} else {
			changes["deactivate_at"] = rec.DeactivateAt
		}
	}

	return changes
}
