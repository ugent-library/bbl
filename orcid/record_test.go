package orcid

import (
	"testing"
)

func TestRecord(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Record(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}
