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
	if !strings.Contains(body, "Create case") {
		t.Fatal("case creation control missing from task board HTML")
	}
	if !strings.Contains(body, "source_report_id") {
		t.Fatal("report lineage handoff missing from task board HTML")
	}
	if !strings.Contains(body, "/api/v1/reports/") {
		t.Fatal("report lookup endpoint missing from task board HTML")
	}
	if !strings.Contains(body, "falling back to task-only case handoff") {
		t.Fatal("report lookup fallback missing from task board HTML")
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
	if !strings.Contains(body, "/admin/trace-detail") {
		t.Fatal("trace detail link missing from task board HTML")
	}
	if !strings.Contains(body, "/admin/cases") {
		t.Fatal("cases page link missing from task board HTML")
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
	if !strings.Contains(body, "/api/v1/reports") {
		t.Fatal("report list API path missing from page HTML")
	}
	if !strings.Contains(body, "/api/v1/tasks/") {
		t.Fatal("task detail API path missing from reports page HTML")
	}
	if !strings.Contains(body, "/api/v1/reports/") {
		t.Fatal("report detail API path missing from reports page HTML")
	}
	if !strings.Contains(body, "/admin/version-detail") {
		t.Fatal("version detail handoff missing from reports page HTML")
	}
	if !strings.Contains(body, "Report Lane") {
		t.Fatal("report lane heading missing from page HTML")
	}
	if !strings.Contains(body, "Open Task Board") {
		t.Fatal("task board handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "Open Cases") {
		t.Fatal("cases handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "/admin/report-compare") {
		t.Fatal("report compare handoff link missing from reports page HTML")
	}
	if !strings.Contains(body, "/admin/trace-detail") {
		t.Fatal("trace detail handoff missing from reports page HTML")
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
	if !strings.Contains(body, "Create case") {
		t.Fatal("report-to-case action missing from reports page HTML")
	}
	if !strings.Contains(body, "source task can be loaded") {
		t.Fatal("report task-provenance fallback guard missing from reports page HTML")
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
	if !strings.Contains(body, "Open report compare") {
		t.Fatal("report compare detail handoff missing from reports page HTML")
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
	if !strings.Contains(body, "Status fixed to ready") {
		t.Fatal("report status summary missing from reports page HTML")
	}
	if !strings.Contains(body, "Type fixed to workflow_summary") {
		t.Fatal("report type summary missing from reports page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected report row styling missing from reports page HTML")
	}
}

func TestAdminEvalsPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/evals")
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
	if !strings.Contains(body, "<title>OpsPilot Eval Cases</title>") {
		t.Fatal("page title missing from eval page HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-cases") {
		t.Fatal("eval case API path missing from eval page HTML")
	}
	if !strings.Contains(body, "Eval Lane") {
		t.Fatal("eval lane heading missing from eval page HTML")
	}
	if !strings.Contains(body, "Copy eval summary") {
		t.Fatal("eval summary handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Copy eval link") {
		t.Fatal("eval link handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Create dataset draft") {
		t.Fatal("dataset draft action missing from eval page HTML")
	}
	if !strings.Contains(body, "Add to dataset") {
		t.Fatal("dataset append action missing from eval page HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-datasets") {
		t.Fatal("eval dataset API path missing from eval page HTML")
	}
	if !strings.Contains(body, "/admin/eval-datasets") {
		t.Fatal("eval dataset lane link missing from eval page HTML")
	}
	if !strings.Contains(body, "Open dataset API detail") {
		t.Fatal("dataset api handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open dataset lane") {
		t.Fatal("dataset lane handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open case API detail") {
		t.Fatal("case handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open report API detail") {
		t.Fatal("report handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open task API detail") {
		t.Fatal("task handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open version detail") {
		t.Fatal("version handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open Trace Detail") {
		t.Fatal("trace detail handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Previous visible") {
		t.Fatal("eval detail navigation missing from eval page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected eval row styling missing from eval page HTML")
	}
}

func TestAdminEvalsPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/evals/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminEvalDatasetsPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-datasets")
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
	if !strings.Contains(body, "<title>OpsPilot Eval Datasets</title>") {
		t.Fatal("page title missing from eval datasets HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-datasets") {
		t.Fatal("eval dataset API path missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Eval Dataset Lane") {
		t.Fatal("eval dataset lane heading missing from page HTML")
	}
	if !strings.Contains(body, "Copy dataset summary") {
		t.Fatal("dataset summary handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Publish dataset") {
		t.Fatal("dataset publish action missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Run dataset") {
		t.Fatal("dataset run action missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "/admin/eval-runs") {
		t.Fatal("eval run lane handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Published datasets are immutable baselines") {
		t.Fatal("dataset published read-only note missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "<option value=\"published\">Published</option>") {
		t.Fatal("published dataset filter missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Copy dataset link") {
		t.Fatal("dataset link handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open dataset API detail") {
		t.Fatal("dataset api detail handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Eval case API detail") {
		t.Fatal("eval case api handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Case API detail") {
		t.Fatal("case api handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Task API detail") {
		t.Fatal("task api handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Report API detail") {
		t.Fatal("report api handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Trace detail") {
		t.Fatal("trace detail handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Version detail") {
		t.Fatal("version detail handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected dataset row styling missing from eval datasets page HTML")
	}
}

func TestAdminEvalDatasetsPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-datasets/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminEvalRunsPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-runs")
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
	if !strings.Contains(body, "<title>OpsPilot Eval Runs</title>") {
		t.Fatal("page title missing from eval runs HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-runs") {
		t.Fatal("eval run API path missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Run Detail") {
		t.Fatal("run detail section missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Copy run summary") {
		t.Fatal("run summary handoff missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Retry run") {
		t.Fatal("retry run action missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Run timeline") {
		t.Fatal("run timeline section missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Run items") {
		t.Fatal("run items section missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open dataset lane") {
		t.Fatal("dataset lane handoff missing from eval runs page HTML")
	}
}

func TestAdminEvalRunsPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-runs/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
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

func TestAdminReportComparePageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/report-compare")
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
	if !strings.Contains(body, "<title>OpsPilot Report Compare</title>") {
		t.Fatal("page title missing from report compare HTML")
	}
	if !strings.Contains(body, "/api/v1/report-compare") {
		t.Fatal("report compare API path missing from page HTML")
	}
	if !strings.Contains(body, "/admin/trace-detail") {
		t.Fatal("trace detail handoff missing from report compare HTML")
	}
	if !strings.Contains(body, "Open left version detail") {
		t.Fatal("left version handoff missing from report compare HTML")
	}
	if !strings.Contains(body, "Open right version detail") {
		t.Fatal("right version handoff missing from report compare HTML")
	}
	if !strings.Contains(body, "/api/v1/reports/") {
		t.Fatal("report detail API path missing from report compare page HTML")
	}
	if !strings.Contains(body, "Load comparison") {
		t.Fatal("load comparison action missing from report compare HTML")
	}
	if !strings.Contains(body, "Swap reports") {
		t.Fatal("swap reports action missing from report compare HTML")
	}
	if !strings.Contains(body, "Comparison summary") {
		t.Fatal("comparison summary section missing from report compare HTML")
	}
}

func TestAdminReportComparePageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/report-compare/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminTraceDetailPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/trace-detail")
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
	if !strings.Contains(body, "<title>OpsPilot Trace Detail</title>") {
		t.Fatal("page title missing from trace detail HTML")
	}
	if !strings.Contains(body, "/api/v1/trace-drilldown") {
		t.Fatal("trace drilldown API path missing from page HTML")
	}
	if !strings.Contains(body, "Load trace detail") {
		t.Fatal("trace detail action missing from page HTML")
	}
	if !strings.Contains(body, "Open Temporal history") {
		t.Fatal("temporal handoff missing from trace detail HTML")
	}
	if !strings.Contains(body, "Open Version Detail") {
		t.Fatal("version handoff missing from trace detail HTML")
	}
}

func TestAdminTraceDetailPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/trace-detail/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminVersionDetailPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/version-detail")
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
	if !strings.Contains(body, "<title>OpsPilot Version Detail</title>") {
		t.Fatal("page title missing from version detail HTML")
	}
	if !strings.Contains(body, "/api/v1/versions") {
		t.Fatal("version API path missing from version detail HTML")
	}
	if !strings.Contains(body, "Copy version summary") {
		t.Fatal("version summary action missing from version detail HTML")
	}
}

func TestAdminVersionDetailPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/version-detail/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminCasesPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/cases")
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
	if !strings.Contains(body, "<title>OpsPilot Cases</title>") {
		t.Fatal("page title missing from admin cases HTML")
	}
	if !strings.Contains(body, "/api/v1/cases") {
		t.Fatal("case list API path missing from page HTML")
	}
	if !strings.Contains(body, "/api/v1/cases/") {
		t.Fatal("case detail API path missing from page HTML")
	}
	if !strings.Contains(body, "Assigned To") {
		t.Fatal("assignee filter missing from cases page HTML")
	}
	if !strings.Contains(body, "Open source task") {
		t.Fatal("source task handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Open source report") {
		t.Fatal("source report handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Close case") {
		t.Fatal("close case action missing from cases page HTML")
	}
	if !strings.Contains(body, "Reopen case") {
		t.Fatal("reopen case action missing from cases page HTML")
	}
	if !strings.Contains(body, "Assign case") {
		t.Fatal("assign case action missing from cases page HTML")
	}
	if !strings.Contains(body, "Add note") {
		t.Fatal("add note action missing from cases page HTML")
	}
	if !strings.Contains(body, "Recent notes") {
		t.Fatal("case notes section missing from cases page HTML")
	}
	if !strings.Contains(body, "My open cases") {
		t.Fatal("my-open-cases quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Unassigned") {
		t.Fatal("unassigned quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Assigned to me") {
		t.Fatal("owned queue summary missing from cases page HTML")
	}
	if !strings.Contains(body, "Task-only") {
		t.Fatal("case provenance badge missing from cases page HTML")
	}
	if !strings.Contains(body, "Open cases") {
		t.Fatal("open-cases quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Age") {
		t.Fatal("age indicator missing from cases page HTML")
	}
	if !strings.Contains(body, "Copy case summary") {
		t.Fatal("case summary handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Promote to eval") {
		t.Fatal("eval promotion action missing from cases page HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-cases") {
		t.Fatal("eval case API path missing from cases page HTML")
	}
	if !strings.Contains(body, "Open eval API detail") {
		t.Fatal("eval api handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Copy case link") {
		t.Fatal("case link handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Open case API detail") {
		t.Fatal("case api handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "/admin/trace-detail") {
		t.Fatal("trace detail handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "<option value=\"closed\">Closed</option>") {
		t.Fatal("closed status filter missing from cases page HTML")
	}
}

func TestAdminCasesPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/cases/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
