package orcid

import (
	"testing"
)

func TestDistinctions(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Distinctions("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
