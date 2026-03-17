package app

import (
	"net/url"
	"strconv"
	"strings"
)

// formGroups parses indexed form fields like prefix[0].field, prefix[1].field
// into a slice of url.Values, one per index. Gaps in indices are skipped.
func formGroups(form url.Values, prefix string) []url.Values {
	var groups []url.Values
	pfx := prefix + "["

	for key, vals := range form {
		rest, ok := strings.CutPrefix(key, pfx)
		if !ok {
			continue
		}
		idxStr, field, ok := strings.Cut(rest, "].")
		if !ok {
			continue
		}
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 {
			continue
		}
		for idx >= len(groups) {
			groups = append(groups, nil)
		}
		if groups[idx] == nil {
			groups[idx] = url.Values{}
		}
		groups[idx][field] = vals
	}

	// Remove nil gaps.
	var result []url.Values
	for _, g := range groups {
		if g != nil {
			result = append(result, g)
		}
	}
	return result
}
