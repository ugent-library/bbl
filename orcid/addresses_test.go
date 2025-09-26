package orcid

import (
	"testing"
)

func TestAddresses(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Addresses(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
