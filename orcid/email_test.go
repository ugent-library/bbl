package orcid

import (
	"testing"
)

func TestEmails(t *testing.T) {
	c := newTestClient()

	// Test with an invalid ORCID id
	if _, _, err := c.Emails("0000-0000-0000-0000"); err != ErrNotFound {
		t.Error("expected ErrNotFound for invalid ORCID id, got nil")
	}

	data, body, err := c.Emails("0000-0003-4791-9455")

	testGet(t, data, body, err)
}
