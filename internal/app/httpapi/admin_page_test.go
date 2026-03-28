package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
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
	if !strings.Contains(body, "Item results") {
		t.Fatal("item results section missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Results</th>") {
		t.Fatal("result summary column missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open dataset lane") {
		t.Fatal("dataset lane handoff missing from eval runs page HTML")
	}
}

func TestAdminEvalReportsPageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-reports")
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
	if !strings.Contains(body, "<title>OpsPilot Eval Reports</title>") {
		t.Fatal("page title missing from eval reports HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-reports") {
		t.Fatal("eval report API path missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Eval Report Lane") {
		t.Fatal("eval report lane heading missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Status fixed to ready") {
		t.Fatal("ready status summary missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Failed items") {
		t.Fatal("failed item summary missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Follow-up") {
		t.Fatal("follow-up summary column missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Needs follow-up") {
		t.Fatal("needs follow-up quick view missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open latest case") {
		t.Fatal("latest case handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Show raw report JSON") {
		t.Fatal("raw report json toggle missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Copy raw report JSON") {
		t.Fatal("raw report json copy action missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Copy report summary") {
		t.Fatal("report summary handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Copy report link") {
		t.Fatal("report link handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Linked cases") {
		t.Fatal("linked cases section missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open linked cases") {
		t.Fatal("linked cases handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "/api/v1/cases") {
		t.Fatal("case list API path missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open eval run lane") {
		t.Fatal("eval run lane handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open dataset API detail") {
		t.Fatal("dataset api handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open eval lane") {
		t.Fatal("eval lane handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "/admin/eval-report-compare") {
		t.Fatal("eval report compare handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Bad cases") {
		t.Fatal("bad cases section missing from eval reports page HTML")
	}
	if !strings.Contains(body, "task-row-selected") {
		t.Fatal("selected eval report row styling missing from eval reports page HTML")
	}
}

func TestAdminEvalReportComparePageRendersHTML(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-report-compare")
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
	if !strings.Contains(body, "<title>OpsPilot Eval Report Compare</title>") {
		t.Fatal("page title missing from eval report compare HTML")
	}
	if !strings.Contains(body, "/api/v1/eval-report-compare") {
		t.Fatal("eval report compare API path missing from page HTML")
	}
	if !strings.Contains(body, "Load comparison") {
		t.Fatal("load comparison action missing from eval report compare HTML")
	}
	if !strings.Contains(body, "Bad-case overlap") {
		t.Fatal("bad-case overlap summary missing from eval report compare HTML")
	}
	if !strings.Contains(body, "Open left eval report API") {
		t.Fatal("left eval report api handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open left latest case") {
		t.Fatal("left latest-case handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right eval report API") {
		t.Fatal("right eval report api handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right latest case") {
		t.Fatal("right latest-case handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Follow-up") {
		t.Fatal("follow-up summary missing from compare HTML")
	}
	if !strings.Contains(body, "Create case from left") {
		t.Fatal("left-side case handoff action missing from compare HTML")
	}
	if !strings.Contains(body, "Create case from right") {
		t.Fatal("right-side case handoff action missing from compare HTML")
	}
	if !strings.Contains(body, "/admin/cases") {
		t.Fatal("cases handoff missing from compare HTML")
	}
	if !strings.Contains(body, "/admin/eval-runs") {
		t.Fatal("eval run handoff missing from compare HTML")
	}
	if !strings.Contains(body, "/admin/version-detail") {
		t.Fatal("version handoff missing from compare HTML")
	}
}

func TestAdminEvalReportComparePageRuntimeSmoke(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-eval-compare-admin-smoke", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare A", "Source Left")
	rightReportID := materializeEvalRunReport(t, "tenant-eval-compare-admin-smoke", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare B", "Source Right")
	leftCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-compare-admin-smoke",
		Title:              "Left compare follow-up",
		Summary:            "left compare summary",
		SourceEvalReportID: leftReportID,
		CreatedBy:          "operator-left",
	})
	if err != nil {
		t.Fatalf("CreateCase(leftCase) error = %v", err)
	}
	rightCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-compare-admin-smoke",
		Title:              "Right compare follow-up",
		Summary:            "right compare summary",
		SourceEvalReportID: rightReportID,
		CreatedBy:          "operator-right",
	})
	if err != nil {
		t.Fatalf("CreateCase(rightCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		EvalReports: reportService,
		Cases:       caseService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "eval_report_compare_smoke.js")
	script := `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const leftReportID = process.argv[4];
const rightReportID = process.argv[5];
const leftCaseID = process.argv[6];
const rightCaseID = process.argv[7];

async function assertCaseSource(page, apiBaseURL, caseID, tenantID, expectedReportID) {
  await page.goto(apiBaseURL + "/api/v1/cases/" + encodeURIComponent(caseID) + "?tenant_id=" + encodeURIComponent(tenantID));
  await page.waitForSelector("body");
  const payload = JSON.parse(await page.textContent("body"));
  if (payload.source_eval_report_id !== expectedReportID) {
    throw new Error("unexpected source_eval_report_id for " + caseID + ": " + payload.source_eval_report_id);
  }
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(tenantID) + "&left_report_id=" + encodeURIComponent(leftReportID) + "&right_report_id=" + encodeURIComponent(rightReportID));
  await page.waitForSelector("text=Comparison summary");
  await page.waitForSelector("text=Bad-case overlap");
  const leftHref = await page.getAttribute("#leftReportAPILink", "href");
  if (!leftHref || !leftHref.includes(leftReportID)) {
    throw new Error("left report API handoff missing selected report");
  }
  const rightHref = await page.getAttribute("#rightReportAPILink", "href");
  if (!rightHref || !rightHref.includes(rightReportID)) {
    throw new Error("right report API handoff missing selected report");
  }
  const leftCaseHref = await page.getAttribute("#leftLatestCaseLink", "href");
  if (!leftCaseHref || !leftCaseHref.includes("case_id=" + encodeURIComponent(leftCaseID))) {
    throw new Error("left latest-case handoff missing selected case");
  }
	const rightCaseHref = await page.getAttribute("#rightLatestCaseLink", "href");
	if (!rightCaseHref || !rightCaseHref.includes("case_id=" + encodeURIComponent(rightCaseID))) {
		throw new Error("right latest-case handoff missing selected case");
	}
  const leftFollowUpText = (await page.textContent("#leftReportDetail")).trim();
  if (!leftFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("left follow-up summary missing from compare detail");
  }
  const rightFollowUpText = (await page.textContent("#rightReportDetail")).trim();
  if (!rightFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("right follow-up summary missing from compare detail");
  }
  await page.click("#createLeftCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const createdLeftURL = new URL(page.url());
  const leftCreatedCaseID = createdLeftURL.searchParams.get("case_id");
  if (!leftCreatedCaseID) {
    throw new Error("left compare-to-case handoff missing case_id");
  }
  if (createdLeftURL.searchParams.get("tenant_id") !== tenantID) {
    throw new Error("left compare-to-case handoff missing tenant_id");
  }
  await assertCaseSource(page, baseURL, leftCreatedCaseID, tenantID, leftReportID);

  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(tenantID) + "&left_report_id=" + encodeURIComponent(leftReportID) + "&right_report_id=" + encodeURIComponent(rightReportID));
  await page.waitForSelector("text=Comparison summary");
  await page.click("#createRightCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const createdRightURL = new URL(page.url());
  const rightCreatedCaseID = createdRightURL.searchParams.get("case_id");
  if (!rightCreatedCaseID) {
    throw new Error("right compare-to-case handoff missing case_id");
  }
  if (createdRightURL.searchParams.get("tenant_id") !== tenantID) {
    throw new Error("right compare-to-case handoff missing tenant_id");
  }
  await assertCaseSource(page, baseURL, rightCreatedCaseID, tenantID, rightReportID);

  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(tenantID) + "&left_report_id=" + encodeURIComponent(leftReportID) + "&right_report_id=missing-report");
  await page.waitForSelector("text=Eval report comparison request failed: 404 Not Found");
  await browser.close();
})().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile(scriptPath) error = %v", err)
	}

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-eval-compare-admin-smoke", leftReportID, rightReportID, leftCase.ID, rightCase.ID)
	cmd.Env = append(os.Environ(), "NODE_PATH="+nodePathRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outText := string(output)
		if strings.Contains(outText, "Please run the following command to download new browsers") ||
			strings.Contains(outText, "Executable doesn't exist") {
			t.Skip("skipping playwright runtime smoke: browser binaries not installed")
		}
		t.Fatalf("playwright runtime smoke failed: %v\n%s", err, string(output))
	}
}

func TestAdminEvalReportsPageRuntimeSmoke(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-admin-smoke", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Follow-up", "Source Follow-up")
	_ = materializeEvalRunReport(t, "tenant-eval-admin-smoke", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset No Follow-up", "Source No Follow-up")

	for i := 0; i < 6; i++ {
		if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:           "tenant-eval-admin-smoke",
			Title:              "Investigate regression",
			Summary:            "Follow up failing eval report",
			SourceEvalReportID: reportID,
			CreatedBy:          "operator-eval",
		}); err != nil {
			t.Fatalf("CreateCase(%d) error = %v", i, err)
		}
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-admin-smoke",
		Title:              "Open regression follow-up",
		Summary:            "Open follow-up summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-eval",
	}); err != nil {
		t.Fatalf("CreateCase(openFollowUp) error = %v", err)
	}
	closedFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-admin-smoke",
		Title:              "Closed regression follow-up",
		Summary:            "Closed follow-up summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateCase(closedFollowUp) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedFollowUp.ID, "operator-eval"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		EvalReports: reportService,
		Cases:       caseService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "eval_reports_smoke.js")
	script := `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const reportID = process.argv[4];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10");
  await page.waitForSelector("text=Eval Report Lane");
  await page.waitForSelector("text=Bad cases");
  await page.waitForSelector("text=Linked cases");
  const visibleCount = (await page.textContent("#visibleCount")).trim();
  if (visibleCount !== "2") {
    throw new Error("unexpected visibleCount before quick view: " + visibleCount);
  }
  await page.click("#quickViewNeedsFollowUp");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("needs_follow_up") === "true");
  const filteredVisibleCount = (await page.textContent("#visibleCount")).trim();
  if (filteredVisibleCount !== "1") {
    throw new Error("unexpected visibleCount after quick view: " + filteredVisibleCount);
  }
  const followUpFilterValue = await page.$eval("#needs_follow_up", (el) => el.value);
  if (followUpFilterValue !== "true") {
    throw new Error("needs_follow_up filter was not synced to quick view");
  }
  const visibleCountAfterQuickView = (await page.textContent("#visibleCount")).trim();
  if (visibleCountAfterQuickView !== "1") {
    throw new Error("unexpected visibleCount: " + visibleCount);
  }
  const followUpSummary = (await page.textContent("#reportRows tr td:nth-child(5)")).trim();
  if (!followUpSummary.includes("8 cases")) {
    throw new Error("follow-up case count missing from list row: " + followUpSummary);
  }
  if (!followUpSummary.includes("7 open")) {
    throw new Error("open follow-up case count missing from list row: " + followUpSummary);
  }
  if (!followUpSummary.includes("latest open")) {
    throw new Error("latest follow-up case status missing from list row: " + followUpSummary);
  }
  const latestCaseHref = await page.getAttribute("#reportRows tr td:nth-child(5) a", "href");
  if (!latestCaseHref || !latestCaseHref.includes("/admin/cases?") || !latestCaseHref.includes("case_id=")) {
    throw new Error("latest case handoff link missing from list row");
  }
  const detailLatestCaseHref = await page.getAttribute("#openLatestCaseLink", "href");
  if (!detailLatestCaseHref || !detailLatestCaseHref.includes("/admin/cases?") || !detailLatestCaseHref.includes("case_id=")) {
    throw new Error("latest case handoff link missing from detail pane");
  }
  const linkedCaseCount = (await page.textContent("#linkedCaseCount")).trim();
  if (linkedCaseCount !== "5+") {
    throw new Error("unexpected linkedCaseCount: " + linkedCaseCount);
  }
  const linkedCaseScope = (await page.textContent("#linkedCasesScope")).trim();
  if (!linkedCaseScope.includes("Showing latest 5")) {
    throw new Error("linked case scope note did not reflect paginated slice: " + linkedCaseScope);
  }
  const linkedCasesHref = await page.getAttribute("#openLinkedCasesLink", "href");
  if (!linkedCasesHref || !linkedCasesHref.includes("source_eval_report_id=" + encodeURIComponent(reportID))) {
    throw new Error("linked cases handoff missing source_eval_report_id");
  }
  const urlAfterLoad = new URL(page.url());
  if (urlAfterLoad.searchParams.get("report_id") !== reportID) {
    throw new Error("selected report_id not synced into URL");
  }

  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&report_id=missing-report");
  await page.waitForSelector("text=Unable to load the selected eval report detail.");
  const failedURL = new URL(page.url());
  if (failedURL.searchParams.get("report_id")) {
    throw new Error("stale report_id remained in URL after detail load failure");
  }
  await browser.close();
})().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile(scriptPath) error = %v", err)
	}

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-eval-admin-smoke", reportID)
	cmd.Env = append(os.Environ(), "NODE_PATH="+nodePathRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outText := string(output)
		if strings.Contains(outText, "Please run the following command to download new browsers") ||
			strings.Contains(outText, "Executable doesn't exist") {
			t.Skip("skipping playwright runtime smoke: browser binaries not installed")
		}
		t.Fatalf("playwright runtime smoke failed: %v\n%s", err, string(output))
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

func TestAdminEvalReportsPageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-reports/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAdminEvalReportComparePageRejectsUnknownSubpath(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/eval-report-compare/unknown")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func npmGlobalRoot() (string, error) {
	cmd := exec.Command("npm.cmd", "root", "-g")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", exec.ErrNotFound
	}
	return root, nil
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
	if !strings.Contains(body, "Open source eval report") {
		t.Fatal("source eval report handoff missing from cases page HTML")
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
	if !strings.Contains(body, "Source eval report") {
		t.Fatal("source eval report detail missing from cases page HTML")
	}
	if !strings.Contains(body, "Open cases") {
		t.Fatal("open-cases quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Eval-backed cases") {
		t.Fatal("eval-backed quick view missing from cases page HTML")
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
	if !strings.Contains(body, "/api/v1/eval-reports/") {
		t.Fatal("eval report api path missing from cases page HTML")
	}
	if !strings.Contains(body, "/admin/eval-reports") {
		t.Fatal("eval reports handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Source eval report summary") {
		t.Fatal("source eval report summary section missing from cases page HTML")
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

func TestAdminCasesPageRuntimeSmoke(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-case-admin-smoke", evalsvc.RunStatusFailed, "failure detail")
	ctx := context.Background()
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}

	caseService := casesvc.NewService()
	linkedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-case-admin-smoke",
		Title:              "Investigate eval regression",
		Summary:            "Follow up eval-linked operator case",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-case",
	})
	if err != nil {
		t.Fatalf("CreateCase(linked) error = %v", err)
	}
	missingCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-case-admin-smoke",
		Title:              "Investigate missing eval report",
		Summary:            "Case should degrade when source eval report is unavailable",
		SourceEvalReportID: "missing-eval-report",
		CreatedBy:          "operator-case",
	})
	if err != nil {
		t.Fatalf("CreateCase(missing) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "cases_smoke.js")
	script := `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const linkedCaseID = process.argv[4];
const missingCaseID = process.argv[5];
const reportID = process.argv[6];
const datasetID = process.argv[7];
const runStatus = process.argv[8];
const summary = process.argv[9];
const badCaseCount = process.argv[10];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/cases?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&case_id=" + encodeURIComponent(linkedCaseID));
  await page.waitForSelector("text=Source eval report summary");
  await page.waitForFunction(
    (expectedReportID) => {
      const node = document.querySelector("#sourceEvalReportSummary");
      return node && node.textContent && node.textContent.includes(expectedReportID);
    },
    reportID
  );
  const detailText = await page.textContent("#sourceEvalReportSummary");
  if (!detailText.includes(datasetID)) throw new Error("missing dataset_id in source eval report summary");
  if (!detailText.includes(runStatus)) throw new Error("missing run_status in source eval report summary");
  if (!detailText.includes(summary)) throw new Error("missing eval report summary text");
  if (!detailText.includes(badCaseCount)) throw new Error("missing bad case count");
  const evalLaneHref = await page.getAttribute("#openEvalReportsLink", "href");
  if (!evalLaneHref || !evalLaneHref.includes("report_id=" + encodeURIComponent(reportID))) {
    throw new Error("eval report lane handoff missing report_id");
  }

  await page.goto(baseURL + "/admin/cases?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&case_id=" + encodeURIComponent(missingCaseID));
  await page.waitForSelector("text=Source eval report summary");
  await page.waitForFunction(() => {
    const node = document.querySelector("#sourceEvalReportSummary");
    return node && node.textContent && node.textContent.includes("Unable to load source eval report metadata");
  });
  const missingEvalAPIHref = await page.getAttribute("#openEvalReportLink", "href");
  if (!missingEvalAPIHref || !missingEvalAPIHref.includes("missing-eval-report")) {
    throw new Error("source eval report API handoff drifted after lookup failure");
  }
  const missingEvalLaneHref = await page.getAttribute("#openEvalReportsLink", "href");
  if (!missingEvalLaneHref || !missingEvalLaneHref.includes("report_id=missing-eval-report")) {
    throw new Error("source eval report lane handoff drifted after lookup failure");
  }
  const failedURL = new URL(page.url());
  if (failedURL.searchParams.get("case_id") !== missingCaseID) {
    throw new Error("selected case_id drifted after source eval report lookup failure");
  }
  await browser.close();
})().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile(scriptPath) error = %v", err)
	}

	cmd := exec.Command(
		"node",
		scriptPath,
		server.URL,
		"tenant-case-admin-smoke",
		linkedCase.ID,
		missingCase.ID,
		reportID,
		reportItem.DatasetID,
		reportItem.RunStatus,
		reportItem.Summary,
		strconv.Itoa(len(reportItem.BadCases)),
	)
	cmd.Env = append(os.Environ(), "NODE_PATH="+nodePathRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outText := string(output)
		if strings.Contains(outText, "Please run the following command to download new browsers") ||
			strings.Contains(outText, "Executable doesn't exist") {
			t.Skip("skipping playwright runtime smoke: browser binaries not installed")
		}
		t.Fatalf("playwright runtime smoke failed: %v\n%s", err, string(output))
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
