package can

import (
	"github.com/ugent-library/bbl"
)

func Curate(u *bbl.User) bool {
	return u.Role == bbl.AdminRole || u.Role == bbl.CuratorRole
}

func ViewWork(u *bbl.User, rec *bbl.Work) bool {
	if rec.Status == bbl.PublicStatus {
		return true
	}
	if u == nil {
		return false
	}
	if u.Role == bbl.AdminRole || u.Role == bbl.CuratorRole {
		return true
	}
	for _, perm := range rec.Permissions {
		if perm.UserID == u.ID && (perm.Kind == bbl.EditPermission || perm.Kind == bbl.ViewPermission) {
			return true
		}
	}
	return false
}

func EditWork(u *bbl.User, rec *bbl.Work) bool {
	if u == nil {
		return false
	}
	if u.Role == bbl.AdminRole || u.Role == bbl.CuratorRole {
		return true
	}
	for _, perm := range rec.Permissions {
		if perm.UserID == u.ID && perm.Kind == bbl.EditPermission {
			return true
		}
	}
	return false
}
