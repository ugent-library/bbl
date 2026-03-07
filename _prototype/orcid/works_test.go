package orcid

import (
	"testing"
)

func TestBulkWorks(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Works(t.Context(), "0000-0003-4791-9455", "1678760")

	testGet(t, data, body, err)
}

func TestWorks(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Works(t.Context(), "0000-0003-4791-9455")

	testGet(t, data, body, err)
}

func TestWork(t *testing.T) {
	c := newTestClient()

	data, body, err := c.Work(t.Context(), "0000-0003-4791-9455", "1678760")

	testGet(t, data, body, err)
}
