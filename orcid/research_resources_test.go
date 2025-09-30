package orcid

import (
	"testing"
)

func TestResearchResources(t *testing.T) {
	c := newTestClient()

	data, body, err := c.ResearchResources(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
