package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opspilot-go/internal/version"
)

func TestGetVersionReturnsCurrentVersion(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Versions: version.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/versions/" + version.DefaultVersionID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body versionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.VersionID != version.DefaultVersionID {
		t.Fatalf("VersionID = %q, want %q", body.VersionID, version.DefaultVersionID)
	}
	if body.PromptBundle == "" {
		t.Fatal("PromptBundle is empty")
	}
}

func TestListVersionsReturnsCurrentVersionPage(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Versions: version.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/versions?limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body listVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(body.Versions) != 1 {
		t.Fatalf("len(Versions) = %d, want %d", len(body.Versions), 1)
	}
	if body.Versions[0].VersionID != version.DefaultVersionID {
		t.Fatalf("Versions[0].VersionID = %q, want %q", body.Versions[0].VersionID, version.DefaultVersionID)
	}
}

func TestGetVersionReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Versions: version.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/versions/version-missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
