package orcid

import (
	"testing"
)

func TestServices(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Services("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
