package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminTaskBoardPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/task-board")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "<title>OpsPilot Task Board</title>") {
		t.Fatal("page title missing from admin task board HTML")
	}
	if !strings.Contains(body, "/api/v1/admin/task-board") {
		t.Fatal("admin task board API path missing from page HTML")
	}
	if !strings.Contains(body, "/api/v1/tasks/") {
		t.Fatal("task detail API path missing from page HTML")
	}
	if !strings.Contains(body, "Task detail") {
		t.Fatal("task detail section missing from page HTML")
	}
	if !strings.Contains(body, "Approve task") {
		t.Fatal("approve task action missing from page HTML")
	}
	if !strings.Contains(body, "Retry task") {
		t.Fatal("retry task action missing from page HTML")
	}
	if !strings.Contains(body, "Temporal execution") {
		t.Fatal("temporal execution panel missing from page HTML")
	}
	if !strings.Contains(body, "Auto refresh") {
		t.Fatal("auto refresh controls missing from page HTML")
	}
}

func TestAdminTaskBoardPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/task-board/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
