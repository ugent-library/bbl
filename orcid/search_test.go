package orcid

import "testing"

func TestSearch(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Search(t.Context(), "Steenlant")

	testGet(t, data, body, err)
}
