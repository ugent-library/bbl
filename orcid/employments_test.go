package orcid

import (
	"testing"
)

func TestEmployments(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Employments("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
