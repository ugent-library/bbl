package orcid

import (
	"testing"
)

func TestKeywords(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Keywords(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
