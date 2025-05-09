package can

import (
	"github.com/ugent-library/bbl"
)

func ViewWork(u *bbl.User, rec *bbl.Work) bool {
	if rec.Status == bbl.PublicStatus {
		return true
	}
	if u.Role == bbl.AdminRole {
		return true
	}
	if u.ID == rec.CreatedByID {
		return true
	}
	return false
}
