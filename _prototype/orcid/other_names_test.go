package orcid

import (
	"testing"
)

func TestOtherNames(t *testing.T) {
	c := newTestClient()

	data, body, err := c.OtherNames(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
