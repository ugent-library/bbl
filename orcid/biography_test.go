package orcid

import (
	"testing"
)

func TestBiography(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Biography(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
