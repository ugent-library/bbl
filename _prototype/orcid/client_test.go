package orcid

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func newTestClient() *Client {
	return NewClient(Config{
		ClientID:     os.Getenv("ORCID_TEST_CLIENT_ID"),
		ClientSecret: os.Getenv("ORCID_TEST_CLIENT_SECRET"),
		Sandbox:      true,
	})
}

func testGet(t *testing.T, data any, body []byte, err error) {
	j, _ := json.MarshalIndent(data, "", "  ")
	t.Logf("body: %s", body)
	t.Logf("data: %s", j)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if data == nil {
		t.Error("expected non-nil data")
	}
	if body == nil {
		t.Error("expected non-nil body")
	}
}

func TestNewClient_Defaults(t *testing.T) {
	cfg := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Sandbox:      false,
	}
	client := NewClient(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != PublicUrl {
		t.Errorf("expected baseURL %s, got %s", PublicUrl, client.baseURL)
	}
	if client.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
}

func TestNewClient_Sandbox(t *testing.T) {
	cfg := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Sandbox:      true,
	}
	client := NewClient(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != SandboxPublicUrl {
		t.Errorf("expected baseURL %s, got %s", SandboxPublicUrl, client.baseURL)
	}
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	customHTTP := &http.Client{}
	cfg := Config{
		HTTPClient: customHTTP,
		Sandbox:    false,
	}
	client := NewClient(cfg)
	if client.httpClient != customHTTP {
		t.Error("expected custom http client to be used")
	}
}
