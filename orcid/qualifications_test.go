package orcid

import (
	"testing"
)

func TestQualifications(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Qualifications("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
