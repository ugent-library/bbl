package cli

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
)

func envStrOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

const filterHelp = `Filter expressions select documents by field values.

Syntax:
  field=value             exact match
  field=val1|val2         match any of the values
  field=a field=b         AND (both must match)
  field=a or field=b      OR (either must match)
  (field=a or field=b)    grouping with parentheses

Examples:
  -f "status=public"
  -f "status=public kind=book|article"
  -f "kind=book or kind=conference_paper"
  -f "status=public (kind=book or kind=article)"
  -f "(status=public and kind=book) or (status=private and kind=article)"`

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
