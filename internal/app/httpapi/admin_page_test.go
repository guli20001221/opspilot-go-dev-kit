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
	if !strings.Contains(body, "Quick views") {
		t.Fatal("quick view controls missing from page HTML")
	}
	if !strings.Contains(body, "Queued") {
		t.Fatal("queued quick view missing from page HTML")
	}
	if !strings.Contains(body, "Succeeded") {
		t.Fatal("succeeded quick view missing from page HTML")
	}
	if !strings.Contains(body, "Succeeded reports") {
		t.Fatal("succeeded-reports quick view missing from page HTML")
	}
	if !strings.Contains(body, "Workflow required") {
		t.Fatal("workflow-required quick view missing from page HTML")
	}
	if !strings.Contains(body, "Approval required") {
		t.Fatal("approval-required quick view missing from page HTML")
	}
	if !strings.Contains(body, "No approval") {
		t.Fatal("no-approval quick view missing from page HTML")
	}
	if !strings.Contains(body, "Failed approvals") {
		t.Fatal("failed-approvals quick view missing from page HTML")
	}
	if !strings.Contains(body, "Report tasks") {
		t.Fatal("report quick view missing from page HTML")
	}
	if !strings.Contains(body, "Approved tools") {
		t.Fatal("approved-tool quick view missing from page HTML")
	}
	if !strings.Contains(body, "Raw JSON") {
		t.Fatal("raw json detail controls missing from page HTML")
	}
	if !strings.Contains(body, "Copy task link") {
		t.Fatal("task handoff controls missing from page HTML")
	}
	if !strings.Contains(body, "Copy audit summary") {
		t.Fatal("audit summary controls missing from page HTML")
	}
	if !strings.Contains(body, "Previous visible") {
		t.Fatal("detail navigation controls missing from page HTML")
	}
	if !strings.Contains(body, "Execution summary") {
		t.Fatal("detail execution summary missing from page HTML")
	}
	if !strings.Contains(body, "Focus same lane") {
		t.Fatal("detail lane focus control missing from page HTML")
	}
	if !strings.Contains(body, "Focus same queue") {
		t.Fatal("detail queue focus control missing from page HTML")
	}
	if !strings.Contains(body, "Focus same task type") {
		t.Fatal("detail task-type focus control missing from page HTML")
	}
	if !strings.Contains(body, "Focus approval lane") {
		t.Fatal("detail approval lane focus control missing from page HTML")
	}
	if !strings.Contains(body, "Focus same reason") {
		t.Fatal("detail reason focus control missing from page HTML")
	}
	if !strings.Contains(body, "Focus same status") {
		t.Fatal("detail status focus control missing from page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected task row styling missing from page HTML")
	}
	if !strings.Contains(body, "/admin/reports") {
		t.Fatal("reports page link missing from task board HTML")
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

func TestAdminReportsPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/reports")
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
	if !strings.Contains(body, "<title>OpsPilot Reports</title>") {
		t.Fatal("page title missing from admin reports HTML")
	}
	if !strings.Contains(body, "/api/v1/admin/task-board") {
		t.Fatal("report lane API path missing from page HTML")
	}
	if !strings.Contains(body, "/api/v1/tasks/") {
		t.Fatal("task detail API path missing from reports page HTML")
	}
	if !strings.Contains(body, "/api/v1/reports/") {
		t.Fatal("report detail API path missing from reports page HTML")
	}
	if !strings.Contains(body, "Report Lane") {
		t.Fatal("report lane heading missing from page HTML")
	}
	if !strings.Contains(body, "Open Task Board") {
		t.Fatal("task board handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "Open current report in Task Board") {
		t.Fatal("current report handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "Copy report summary") {
		t.Fatal("report summary handoff action missing from reports page HTML")
	}
	if !strings.Contains(body, "Copy report link") {
		t.Fatal("report link handoff action missing from reports page HTML")
	}
	if !strings.Contains(body, "Show raw report JSON") {
		t.Fatal("report raw json toggle missing from reports page HTML")
	}
	if !strings.Contains(body, "Copy raw report JSON") {
		t.Fatal("report raw json copy action missing from reports page HTML")
	}
	if !strings.Contains(body, "Open report API detail") {
		t.Fatal("report api handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "Report ID") {
		t.Fatal("report identity section missing from reports page HTML")
	}
	if !strings.Contains(body, "Previous visible") {
		t.Fatal("report detail navigation controls missing from reports page HTML")
	}
	if !strings.Contains(body, "Auto refresh") {
		t.Fatal("report lane auto refresh controls missing from reports page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected report row styling missing from reports page HTML")
	}
}

func TestAdminReportsPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/reports/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
