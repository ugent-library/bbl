package orcid

import (
	"testing"
)

func TestMemberships(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Memberships(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
