package bbl

import "slices"

// dedup returns ids with duplicates removed, preserving order.
func dedup(ids []ID) []ID {
	seen := make(map[ID]struct{}, len(ids))
	out := make([]ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// slicesEqual reports whether two slices of comparable elements are equal.
func slicesEqual[T comparable](a, b []T) bool {
	return slices.Equal(a, b)
}

// idPtrEqual reports whether two *ID pointers are equal (both nil or same value).
func idPtrEqual(a, b *ID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// contributorsEqual compares two WorkContributor slices for equality.
func contributorsEqual(a, b []WorkContributor) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind || a[i].Name != b[i].Name ||
			a[i].GivenName != b[i].GivenName || a[i].FamilyName != b[i].FamilyName ||
			!idPtrEqual(a[i].PersonID, b[i].PersonID) ||
			!slicesEqual(a[i].Roles, b[i].Roles) {
			return false
		}
	}
	return true
}

// personOrganizationsEqual compares two PersonOrganization slices for equality.
func personOrganizationsEqual(a, b []PersonOrganization) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].OrganizationID != b[i].OrganizationID || a[i].Role != b[i].Role {
			return false
		}
	}
	return true
}

// projectPeopleEqual compares two ProjectPerson slices for equality.
func projectPeopleEqual(a, b []ProjectPerson) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].PersonID != b[i].PersonID || a[i].Role != b[i].Role {
			return false
		}
	}
	return true
}

// workRelsMatch compares cached WorkRel slice against a slice of items
// that have RelatedWorkID and Kind fields.
func workRelsMatch(cached []WorkRel, input []struct {
	RelatedWorkID ID     `json:"related_work_id"`
	Kind          string `json:"kind"`
}) bool {
	if len(cached) != len(input) {
		return false
	}
	for i := range cached {
		if cached[i].RelatedWorkID != input[i].RelatedWorkID || cached[i].Kind != input[i].Kind {
			return false
		}
	}
	return true
}

// orgRelsMatch compares cached OrganizationRel slice against the anonymous
// struct slice used by SetOrganizationRels.
func orgRelsMatch(cached []OrganizationRel, input []struct {
	RelOrganizationID ID     `json:"rel_organization_id"`
	Kind              string `json:"kind"`
}) bool {
	if len(cached) != len(input) {
		return false
	}
	for i := range cached {
		if cached[i].RelOrganizationID != input[i].RelOrganizationID || cached[i].Kind != input[i].Kind {
			return false
		}
	}
	return true
}
