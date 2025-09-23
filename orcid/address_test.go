package orcid

import (
	"testing"
)

func TestAddress(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Address("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
