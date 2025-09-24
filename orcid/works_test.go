package orcid

import (
	"testing"
)

func TestWorks(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Works("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
