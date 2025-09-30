package bbl

import (
	"time"
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
	return nil
	// v := valgo.New()
	// v.Is(
	// 	valgo.String(rec.Username, "username").Not().Blank(),
	// 	valgo.String(rec.Username, "email").Not().Blank(),
	// 	valgo.String(rec.Name, "name").Not().Blank(),
	// 	valgo.String(rec.Name, "name").Not().Blank(),
	// 	valgo.String(rec.Role, "role").InSlice(UserRoles),
	// )
	// for i, ident := range rec.Identifiers {
	// 	v.InRow("identifiers", i, v.Is(
	// 		valgo.String(ident.Scheme, "scheme").Not().Blank(),
	// 		valgo.String(ident.Val, "val").Not().Blank(),
	// 	))
	// }
	// return v.ToError()
}
