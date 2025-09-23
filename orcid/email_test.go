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

	data, res, err := c.Emails("0000-0003-4791-9455")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data == nil {
		t.Error("expected non-nil Emails data")
	}
	if res == nil {
		t.Error("expected non-nil http.Response")
	}
}
