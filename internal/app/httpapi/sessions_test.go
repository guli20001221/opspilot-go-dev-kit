package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSessionEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","user_id":"user-1"}`)
	resp, err := http.Post(server.URL+"/api/v1/sessions", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got["session_id"] == "" {
		t.Fatal("session_id is empty")
	}
}

func TestListSessionMessagesEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	sessionID := createSessionForTest(t, server.URL)

	resp, err := http.Get(server.URL + "/api/v1/sessions/" + sessionID + "/messages")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got struct {
		Messages []map[string]string `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Messages) != 0 {
		t.Fatalf("len(Messages) = %d, want %d", len(got.Messages), 0)
	}
}

func TestUnknownSessionReturnsJSONError(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/sessions/missing/messages")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got["code"] != "session_not_found" {
		t.Fatalf("code = %v, want %q", got["code"], "session_not_found")
	}
}

func createSessionForTest(t *testing.T, baseURL string) string {
	t.Helper()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","user_id":"user-1"}`)
	resp, err := http.Post(baseURL+"/api/v1/sessions", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	return got["session_id"]
}
