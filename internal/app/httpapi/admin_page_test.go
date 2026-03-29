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
	"time"

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
	if !strings.Contains(body, "Create case") {
		t.Fatal("eval case handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open existing case") {
		t.Fatal("existing eval case reuse action missing from eval page HTML")
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
	if !strings.Contains(body, "Open latest follow-up case") {
		t.Fatal("latest follow-up case handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open queue") {
		t.Fatal("row-level eval queue handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Needs follow-up") {
		t.Fatal("needs follow-up quick view missing from eval page HTML")
	}
	if !strings.Contains(body, "needs_follow_up") {
		t.Fatal("needs_follow_up filter missing from eval page HTML")
	}
	if !strings.Contains(body, "Open follow-up slice") {
		t.Fatal("follow-up slice handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Open report API detail") {
		t.Fatal("report handoff missing from eval page HTML")
	}
	if !strings.Contains(body, "Follow-up cases") {
		t.Fatal("follow-up summary missing from eval page HTML")
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

func TestAdminEvalsPageRuntimeSmoke(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)

	sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:       "tenant-eval-page-smoke",
		Title:          "Eval source case",
		Summary:        "Eval source summary",
		SourceTaskID:   "task-eval-page-smoke",
		SourceReportID: "report-eval-page-smoke",
		CreatedBy:      "operator-eval-page",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(context.Background(), evalsvc.CreateInput{
		TenantID:     sourceCase.TenantID,
		SourceCaseID: sourceCase.ID,
		OperatorNote: "promote for eval page smoke",
		CreatedBy:    "operator-eval-page",
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	openFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         sourceCase.TenantID,
		Title:            "Open eval follow-up",
		Summary:          "Open eval follow-up summary",
		SourceEvalCaseID: evalCase.ID,
		CreatedBy:        "operator-eval-page",
	})
	if err != nil {
		t.Fatalf("CreateCase(openFollowUp) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalCaseService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	const script = `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const evalCaseID = process.argv[4];
const latestFollowUpID = process.argv[5];

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/evals?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10");
  await page.waitForSelector("#evalRows tr");
  const followUpSummary = (await page.textContent("#evalRows tr td:nth-child(4)")).trim();
  if (!followUpSummary.includes("1 cases / 1 open")) {
    throw new Error("follow-up summary missing from eval row: " + followUpSummary);
  }
  if (!followUpSummary.includes("Open existing case") || !followUpSummary.includes("Open queue")) {
    throw new Error("row-level follow-up handoff missing from eval row: " + followUpSummary);
  }
  const rowLatestCaseHref = await page.getAttribute("#evalRows tr td:nth-child(4) a[href*='case_id=']", "href");
  if (!rowLatestCaseHref || !rowLatestCaseHref.includes("case_id=" + encodeURIComponent(latestFollowUpID))) {
    throw new Error("row-level latest follow-up case handoff missing");
  }
  const rowQueueHref = await page.locator("#evalRows tr td:nth-child(4) a").evaluateAll((elements) => {
    const match = elements.find((element) => element.textContent && element.textContent.includes("Open queue"));
    return match ? match.getAttribute("href") : "";
  });
  if (!rowQueueHref || !rowQueueHref.includes("source_eval_case_id=" + encodeURIComponent(evalCaseID))) {
    throw new Error("row-level eval queue handoff missing");
  }
  const primaryAction = (await page.textContent("#createCaseButton")).trim();
  if (primaryAction !== "Open existing case") {
    throw new Error("primary eval case action did not switch to existing case reuse: " + primaryAction);
  }
  const actionMode = await page.getAttribute("#createCaseButton", "data-action");
  if (actionMode !== "open-existing") {
    throw new Error("eval case action mode missing reuse state: " + actionMode);
  }
  const targetHref = await page.getAttribute("#createCaseButton", "data-target-href");
  if (!targetHref || !targetHref.includes("case_id=" + encodeURIComponent(latestFollowUpID))) {
    throw new Error("eval case reuse target missing canonical case handoff");
  }
  const latestFollowUpHref = await page.getAttribute("a[href*='case_id=" + encodeURIComponent(latestFollowUpID) + "']", "href");
  if (!latestFollowUpHref || !latestFollowUpHref.includes("/admin/cases?")) {
    throw new Error("latest follow-up handoff missing from eval detail");
  }
  const detailText = (await page.textContent("#evalDetail")).trim();
  if (!detailText.includes(evalCaseID) || !detailText.includes("Follow-up cases")) {
    throw new Error("eval detail did not materialize expected content");
  }
  const statusText = (await page.textContent("#evalDetailStatusNote")).trim();
  if (!statusText.includes("already has open follow-up work")) {
    throw new Error("eval detail status note did not explain case reuse");
  }
  await browser.close();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
`

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "admin-evals-smoke.js")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile(scriptPath) error = %v", err)
	}

	cmd := exec.Command("node", scriptPath, server.URL, sourceCase.TenantID, evalCase.ID, openFollowUp.ID)
	cmd.Env = append(os.Environ(), "NODE_PATH="+nodePathRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Executable doesn't exist") {
			t.Skipf("skipping playwright runtime smoke: %s", strings.TrimSpace(string(output)))
		}
		t.Fatalf("admin evals runtime smoke failed: %v\n%s", err, output)
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
	if !strings.Contains(body, "Needs follow-up") {
		t.Fatal("needs-follow-up quick view missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "needs_follow_up") {
		t.Fatal("needs_follow_up filter missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open latest run") {
		t.Fatal("latest run handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open latest report") {
		t.Fatal("latest report handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open preferred queue") {
		t.Fatal("preferred follow-up queue handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open latest dataset case") {
		t.Fatal("latest dataset case handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Open follow-up cases") {
		t.Fatal("follow-up cases handoff missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Recent eval activity") {
		t.Fatal("recent eval activity panel missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Unresolved follow-up items") {
		t.Fatal("unresolved follow-up summary missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Dataset-wide follow-up case summary") {
		t.Fatal("dataset-wide follow-up case summary panel missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Linked dataset case summary") {
		t.Fatal("linked dataset case summary panel missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Latest-report follow-up case summary") {
		t.Fatal("latest-report follow-up case summary panel missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "Closed follow-up cases") {
		t.Fatal("closed follow-up case summary missing from eval datasets page HTML")
	}
	if !strings.Contains(body, "source_eval_dataset_id") {
		t.Fatal("dataset-scoped case queue handoff missing from eval datasets page HTML")
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

func TestAdminEvalDatasetsPageRuntimeSmoke(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	olderSourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-admin-smoke",
		Title:    "Older dataset source",
	})
	if err != nil {
		t.Fatalf("CreateCase(older) error = %v", err)
	}
	olderEvalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-admin-smoke",
		SourceCaseID: olderSourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(older) error = %v", err)
	}
	if _, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-admin-smoke",
		Name:        "Dataset Without Run",
		EvalCaseIDs: []string{olderEvalCase.ID},
		CreatedBy:   "operator-dataset",
	}); err != nil {
		t.Fatalf("CreateDataset(older) error = %v", err)
	}

	reportID := materializeEvalRunReport(t, "tenant-dataset-admin-smoke", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset With Run", "Dataset With Run Source")
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-admin-smoke",
		Title:              "Dataset admin follow-up",
		SourceEvalReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
	}
	followUpCase, err = caseService.AssignCase(ctx, followUpCase, "dataset-smoke-operator")
	if err != nil {
		t.Fatalf("AssignCase(follow-up) error = %v", err)
	}
	runBackedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:        "tenant-dataset-admin-smoke",
		Title:           "Dataset admin run-backed follow-up",
		SourceEvalRunID: reportItem.RunID,
	})
	if err != nil {
		t.Fatalf("CreateCase(run-backed) error = %v", err)
	}
	runBackedCase, err = caseService.AssignCase(ctx, runBackedCase, "dataset-run-operator")
	if err != nil {
		t.Fatalf("AssignCase(run-backed) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "eval_datasets_smoke.js")
	script := `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const datasetID = process.argv[4];
const runID = process.argv[5];
const reportID = process.argv[6];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/eval-datasets?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10");
  await page.waitForSelector("#datasetRows tr");
    const firstRowText = await page.textContent("#datasetRows tr:first-child");
    if (!firstRowText.includes(runID)) throw new Error("latest run summary missing from dataset row");
    if (!firstRowText.includes("1 unresolved follow-up items")) throw new Error("unresolved follow-up count missing from dataset row");
    if (!firstRowText.includes("1 total / 1 open / 0 closed dataset follow-up cases")) throw new Error("dataset-wide follow-up summary missing from dataset row");
    if (!firstRowText.includes("Latest dataset case: ` + followUpCase.ID + ` (open, assigned to dataset-smoke-operator)")) throw new Error("linked dataset case summary missing from dataset row");
    if (!firstRowText.includes("Latest run-backed case: ` + runBackedCase.ID + ` (open, assigned to dataset-run-operator)")) throw new Error("run-backed case summary missing from dataset row");
    if (!firstRowText.includes("Open latest dataset case")) throw new Error("latest dataset case handoff missing from dataset row");
    if (!firstRowText.includes("Open latest run-backed case")) throw new Error("latest run-backed case handoff missing from dataset row");
    const latestRunHref = await page.getAttribute('a[href*="/admin/eval-runs?"][href*="run_id=' + encodeURIComponent(runID) + '"]', "href");
    if (!latestRunHref) throw new Error("latest run handoff missing from dataset row");
    const latestReportHref = await page.getAttribute('a[href*="/admin/eval-reports?"][href*="selected_report_id=' + encodeURIComponent(reportID) + '"]', "href");
    if (!latestReportHref) throw new Error("latest report handoff missing from dataset row");
    const datasetRowCaseHref = await page.getAttribute('#datasetRows tr:first-child a[href*="/admin/cases?"][href*="case_id=' + encodeURIComponent("` + followUpCase.ID + `") + '"]', "href");
    if (!datasetRowCaseHref) throw new Error("latest dataset case handoff missing from dataset row");
  const detailText = await page.textContent("#datasetDetail");
  if (!detailText.includes(runID)) throw new Error("latest run summary missing from dataset detail");
  if (!detailText.includes(reportID)) throw new Error("latest report summary missing from dataset detail");
    if (!detailText.includes("Dataset-wide follow-up case summary")) throw new Error("dataset-wide follow-up case summary section missing from dataset detail");
    if (!detailText.includes("Linked dataset case summary")) throw new Error("linked dataset case summary section missing from dataset detail");
  if (!detailText.includes("Run-backed case summary")) throw new Error("run-backed case summary section missing from dataset detail");
  if (!detailText.includes("Latest-report follow-up case summary")) throw new Error("latest-report follow-up case summary section missing from dataset detail");
  if (!detailText.includes("Total follow-up cases")) throw new Error("follow-up case total missing from dataset detail");
  if (!detailText.includes("Closed follow-up cases")) throw new Error("closed follow-up case count missing from dataset detail");
  if (!detailText.includes("dataset-smoke-operator")) throw new Error("linked dataset case owner missing from dataset detail");
  if (!detailText.includes("dataset-run-operator")) throw new Error("run-backed case owner missing from dataset detail");
  if (!detailText.includes("Run-backed cases: 1 total / 1 open")) throw new Error("recent activity run-backed case summary missing from dataset detail");
  const datasetCaseHref = await page.getAttribute('a[href*="/admin/cases?"][href*="case_id=' + encodeURIComponent("` + followUpCase.ID + `") + '"]', "href");
  if (!datasetCaseHref) throw new Error("linked dataset case handoff missing from dataset detail");
  const runBackedCaseHref = await page.getAttribute('a[href*="/admin/cases?"][href*="case_id=' + encodeURIComponent("` + runBackedCase.ID + `") + '"]', "href");
  if (!runBackedCaseHref) throw new Error("run-backed case handoff missing from dataset detail");
  const recentRunBackedCaseHref = await page.getAttribute('a[href*="/admin/cases?"][href*="case_id=' + encodeURIComponent("` + runBackedCase.ID + `") + '"]', "href");
  if (!recentRunBackedCaseHref) throw new Error("recent activity run-backed case handoff missing from dataset detail");
  await page.click("#needsFollowUpQuickView");
  await page.waitForFunction(() => document.querySelector("#visibleCount")?.textContent?.trim() === "1");
  const currentURL = new URL(page.url());
  if (currentURL.searchParams.get("needs_follow_up") !== "true") throw new Error("needs_follow_up filter not synced to URL");
  const filterValue = await page.$eval("#needs_follow_up", (node) => node.value);
  if (filterValue !== "true") throw new Error("needs_follow_up filter control not synced");
  const selectedRow = await page.getAttribute('[data-dataset-row="' + datasetID + '"]', "aria-current");
  if (selectedRow !== "true") throw new Error("selected dataset row did not stay synced");
  await browser.close();
})().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile(scriptPath) error = %v", err)
	}

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-dataset-admin-smoke", reportItem.DatasetID, reportItem.RunID, reportID)
	cmd.Env = append(os.Environ(), "NODE_PATH="+nodePathRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outText := string(output)
		if strings.Contains(outText, "Please run the following command to download new browsers") ||
			strings.Contains(outText, "Executable doesn't exist") {
			t.Skip("skipping playwright runtime smoke: browser binaries not installed")
		}
		t.Fatalf("playwright runtime smoke failed: %v\n%s", err, outText)
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
	if !strings.Contains(body, "Needs follow-up") {
		t.Fatal("needs-follow-up quick view missing from eval runs page HTML")
	}
	if !strings.Contains(body, "needs_follow_up") {
		t.Fatal("needs_follow_up filter missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Unresolved failed items") {
		t.Fatal("unresolved failed-item summary missing from eval runs page HTML")
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
	if !strings.Contains(body, "Open eval report") {
		t.Fatal("eval report handoff missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open eval report API detail") {
		t.Fatal("eval report api handoff missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open latest run case") {
		t.Fatal("latest run case handoff missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Eval report ID") {
		t.Fatal("eval report identity missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Linked run case summary") {
		t.Fatal("linked run case summary missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Create case from result") {
		t.Fatal("result-level create-case action missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open existing case") {
		t.Fatal("canonical follow-up reuse action missing from eval runs page HTML")
	}
	if !strings.Contains(body, "Open follow-up slice") {
		t.Fatal("eval run follow-up slice handoff missing from eval runs page HTML")
	}
}

func TestAdminEvalRunsPageRuntimeSmoke(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	makeFailedRunWithFollowUp := func(title string, closeFollowUp bool) (evalsvc.EvalRun, casesvc.Case) {
		t.Helper()
		sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:  "tenant-eval-run-admin-smoke",
			Title:     title + " source",
			Summary:   title + " source summary",
			CreatedBy: "operator-eval-run",
		})
		if err != nil {
			t.Fatalf("CreateCase(sourceCase) error = %v", err)
		}
		evalCase, _, err := evalCaseService.PromoteCase(context.Background(), evalsvc.CreateInput{
			TenantID:     sourceCase.TenantID,
			SourceCaseID: sourceCase.ID,
			CreatedBy:    "operator-eval-run",
		})
		if err != nil {
			t.Fatalf("PromoteCase() error = %v", err)
		}
		dataset, err := datasetService.CreateDataset(context.Background(), evalsvc.CreateDatasetInput{
			TenantID:    sourceCase.TenantID,
			Name:        title + " dataset",
			EvalCaseIDs: []string{evalCase.ID},
		})
		if err != nil {
			t.Fatalf("CreateDataset() error = %v", err)
		}
		if _, err := datasetService.PublishDataset(context.Background(), dataset.ID, evalsvc.PublishDatasetInput{
			TenantID: sourceCase.TenantID,
		}); err != nil {
			t.Fatalf("PublishDataset() error = %v", err)
		}
		run, err := runService.CreateRun(context.Background(), evalsvc.CreateRunInput{
			TenantID:  sourceCase.TenantID,
			DatasetID: dataset.ID,
		})
		if err != nil {
			t.Fatalf("CreateRun() error = %v", err)
		}
		if _, err := runService.ClaimQueuedRuns(context.Background(), 10); err != nil {
			t.Fatalf("ClaimQueuedRuns() error = %v", err)
		}
		if _, err := runService.MarkRunFailed(context.Background(), run.ID, "fault injection: eval run failed"); err != nil {
			t.Fatalf("MarkRunFailed() error = %v", err)
		}
		followUpCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:         sourceCase.TenantID,
			Title:            title + " follow-up",
			Summary:          title + " follow-up summary",
			SourceEvalCaseID: evalCase.ID,
			CreatedBy:        "operator-eval-run",
		})
		if err != nil {
			t.Fatalf("CreateCase(followUpCase) error = %v", err)
		}
		if closeFollowUp {
			if _, err := caseService.CloseCase(context.Background(), followUpCase.ID, "operator-eval-run"); err != nil {
				t.Fatalf("CloseCase(followUpCase) error = %v", err)
			}
		}
		return run, followUpCase
	}

	openRun, openCase := makeFailedRunWithFollowUp("Open latest case", false)
	closedRun, closedCase := makeFailedRunWithFollowUp("Closed latest case", true)

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	nodePathRoot, err := npmGlobalRoot()
	if err != nil {
		t.Skipf("skipping playwright runtime smoke: %v", err)
	}

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "eval-runs-runtime-smoke.js")
	const script = `
const { chromium } = require("playwright");
const baseURL = process.argv[2];
const tenantID = process.argv[3];
const openRunID = process.argv[4];
const openCaseID = process.argv[5];
const closedRunID = process.argv[6];
const closedCaseID = process.argv[7];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(baseURL + "/admin/eval-runs?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&run_id=" + encodeURIComponent(closedRunID));
  await page.waitForSelector("#runRows tr");

  const openRowHref = await page.getAttribute('[data-run-row="' + openRunID + '"] a[href*="case_id=' + encodeURIComponent(openCaseID) + '"]', "href");
  if (!openRowHref) throw new Error("open latest run case handoff missing from open row");

  const closedRowHref = await page.getAttribute('[data-run-row="' + closedRunID + '"] a[href*="case_id=' + encodeURIComponent(closedCaseID) + '"]', "href");
  if (closedRowHref) throw new Error("closed latest run case handoff should be suppressed in row");

  const detailText = await page.textContent("#runDetail");
  if (!detailText.includes(closedCaseID)) throw new Error("closed linked case summary missing from detail");
  if (!detailText.toLowerCase().includes("closed")) throw new Error("closed linked case status missing from detail");

  const detailLinks = await page.locator('#runDetail a').evaluateAll((elements) => elements.map((element) => ({
    text: (element.textContent || "").trim(),
    href: element.getAttribute("href") || ""
  })));
  if (detailLinks.some((entry) => entry.text === "Open latest run case" && entry.href.includes("case_id=" + encodeURIComponent(closedCaseID)))) {
    throw new Error("closed latest run case handoff should be suppressed in detail");
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

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-eval-run-admin-smoke", openRun.ID, openCase.ID, closedRun.ID, closedCase.ID)
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
	if !strings.Contains(body, "Unresolved bad cases") {
		t.Fatal("unresolved bad cases quick view missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open latest case") {
		t.Fatal("latest case handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open unresolved bad cases") {
		t.Fatal("unresolved bad-case row handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "selected_report_id") {
		t.Fatal("selected_report_id URL state missing from eval reports page HTML")
	}
	if !strings.Contains(body, "bad_case_needs_follow_up") {
		t.Fatal("bad_case_needs_follow_up filter missing from eval reports page HTML")
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
	if !strings.Contains(body, "Create case") {
		t.Fatal("create-case action missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Create case from bad case") {
		t.Fatal("bad-case create-case action missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open latest bad-case case") {
		t.Fatal("bad-case latest case handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open bad-case follow-up slice") {
		t.Fatal("bad-case follow-up slice handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Bad cases needing follow-up") {
		t.Fatal("bad-case needs-follow-up quick view missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Bad cases without follow-up") {
		t.Fatal("bad-case no-follow-up quick view missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Linked cases") {
		t.Fatal("linked cases section missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open linked cases") {
		t.Fatal("linked cases handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open compare queue") {
		t.Fatal("compare queue handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open compare follow-ups") {
		t.Fatal("compare follow-ups handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open latest compare-origin case") {
		t.Fatal("latest compare-origin case handoff missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open existing case") {
		t.Fatal("existing report case reuse action missing from eval reports page HTML")
	}
	if !strings.Contains(body, "Open existing bad-case case") {
		t.Fatal("existing bad-case case reuse action missing from eval reports page HTML")
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
	if !strings.Contains(body, "Open left linked cases") {
		t.Fatal("left linked-cases handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open left compare follow-ups") {
		t.Fatal("left compare-follow-ups handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right unresolved bad cases") {
		t.Fatal("unresolved bad-case compare handoff missing from HTML")
	}
	if !strings.Contains(body, "Open right eval report API") {
		t.Fatal("right eval report api handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right latest case") {
		t.Fatal("right latest-case handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right linked cases") {
		t.Fatal("right linked-cases handoff missing from compare HTML")
	}
	if !strings.Contains(body, "Open right compare follow-ups") {
		t.Fatal("right compare-follow-ups handoff missing from compare HTML")
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
	createOnlyLeftReportID := materializeEvalRunReport(t, "tenant-eval-compare-admin-create", evalsvc.RunStatusSucceeded, "create success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Create A", "Source Create Left")
	createOnlyRightReportID := materializeEvalRunReport(t, "tenant-eval-compare-admin-create", evalsvc.RunStatusFailed, "create failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Create B", "Source Create Right")
	leftCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-compare-admin-smoke",
		Title:              "Left compare follow-up",
		Summary:            "left compare summary",
		SourceEvalReportID: leftReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "left",
		},
		CreatedBy: "operator-left",
	})
	if err != nil {
		t.Fatalf("CreateCase(leftCase) error = %v", err)
	}
	rightCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-compare-admin-smoke",
		Title:              "Right compare follow-up",
		Summary:            "right compare summary",
		SourceEvalReportID: rightReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "right",
		},
		CreatedBy: "operator-right",
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
const createOnlyTenantID = process.argv[8];
const createOnlyLeftReportID = process.argv[9];
const createOnlyRightReportID = process.argv[10];

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
  const leftLinkedCasesHref = await page.getAttribute("#leftLinkedCasesLink", "href");
  if (!leftLinkedCasesHref || !leftLinkedCasesHref.includes("/admin/cases?") || !leftLinkedCasesHref.includes("source_eval_report_id=" + encodeURIComponent(leftReportID))) {
    throw new Error("left linked-cases handoff missing selected source_eval_report_id");
  }
  const leftCompareCasesHref = await page.getAttribute("#leftCompareCasesLink", "href");
  if (!leftCompareCasesHref || !leftCompareCasesHref.includes("/admin/cases?") || !leftCompareCasesHref.includes("source_eval_report_id=" + encodeURIComponent(leftReportID)) || !leftCompareCasesHref.includes("compare_origin_only=true") || !leftCompareCasesHref.includes("status=open")) {
    throw new Error("left compare-follow-ups handoff missing canonical compare queue filter");
  }
  const leftPrimaryText = (await page.textContent("#createLeftCaseButton")).trim();
  if (leftPrimaryText !== "Open left compare queue") {
    throw new Error("left primary action should switch to compare queue when open compare follow-up exists");
  }
  const leftBadCaseNeedsFollowUpVisible = await page.isVisible("#leftBadCaseNeedsFollowUpLink");
  if (leftBadCaseNeedsFollowUpVisible) {
    throw new Error("left unresolved bad-case handoff should stay hidden when there are no uncovered bad cases");
  }
	const rightCaseHref = await page.getAttribute("#rightLatestCaseLink", "href");
	if (!rightCaseHref || !rightCaseHref.includes("case_id=" + encodeURIComponent(rightCaseID))) {
		throw new Error("right latest-case handoff missing selected case");
	}
  const rightLinkedCasesHref = await page.getAttribute("#rightLinkedCasesLink", "href");
  if (!rightLinkedCasesHref || !rightLinkedCasesHref.includes("/admin/cases?") || !rightLinkedCasesHref.includes("source_eval_report_id=" + encodeURIComponent(rightReportID))) {
    throw new Error("right linked-cases handoff missing selected source_eval_report_id");
  }
  const rightCompareCasesHref = await page.getAttribute("#rightCompareCasesLink", "href");
  if (!rightCompareCasesHref || !rightCompareCasesHref.includes("/admin/cases?") || !rightCompareCasesHref.includes("source_eval_report_id=" + encodeURIComponent(rightReportID)) || !rightCompareCasesHref.includes("compare_origin_only=true") || !rightCompareCasesHref.includes("status=open")) {
    throw new Error("right compare-follow-ups handoff missing canonical compare queue filter");
  }
  const rightPrimaryText = (await page.textContent("#createRightCaseButton")).trim();
  if (rightPrimaryText !== "Open right compare queue") {
    throw new Error("right primary action should switch to compare queue when open compare follow-up exists");
  }
  const rightBadCaseNeedsFollowUpHref = await page.getAttribute("#rightBadCaseNeedsFollowUpLink", "href");
  if (!rightBadCaseNeedsFollowUpHref || !rightBadCaseNeedsFollowUpHref.includes("/admin/eval-reports?") || !rightBadCaseNeedsFollowUpHref.includes("bad_case_needs_follow_up=true") || !rightBadCaseNeedsFollowUpHref.includes("report_id=" + encodeURIComponent(rightReportID)) || !rightBadCaseNeedsFollowUpHref.includes("selected_report_id=" + encodeURIComponent(rightReportID))) {
    throw new Error("right unresolved bad-case handoff missing canonical eval-report filter");
  }
  const leftFollowUpText = (await page.textContent("#leftReportDetail")).trim();
  if (!leftFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("left follow-up summary missing from compare detail");
  }
  if (!leftFollowUpText.includes("Compare follow-up") || !leftFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("left compare-derived summary missing from compare detail");
  }
  const rightFollowUpText = (await page.textContent("#rightReportDetail")).trim();
  if (!rightFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("right follow-up summary missing from compare detail");
  }
  if (!rightFollowUpText.includes("Compare follow-up") || !rightFollowUpText.includes("1 cases / 1 open")) {
    throw new Error("right compare-derived summary missing from compare detail");
  }
  if (!rightFollowUpText.includes("1 uncovered")) {
    throw new Error("right uncovered bad-case summary missing from compare detail");
  }
  await page.click("#createLeftCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const leftQueueURL = new URL(page.url());
  if (leftQueueURL.searchParams.get("source_eval_report_id") !== leftReportID) {
    throw new Error("left compare queue handoff missing source_eval_report_id");
  }
  if (leftQueueURL.searchParams.get("tenant_id") !== tenantID) {
    throw new Error("left compare queue handoff missing tenant_id");
  }
  if (leftQueueURL.searchParams.get("compare_origin_only") !== "true" || leftQueueURL.searchParams.get("status") !== "open") {
    throw new Error("left compare queue handoff missing canonical compare queue filters");
  }

  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(tenantID) + "&left_report_id=" + encodeURIComponent(leftReportID) + "&right_report_id=" + encodeURIComponent(rightReportID));
  await page.waitForSelector("text=Comparison summary");
  await page.click("#createRightCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const rightQueueURL = new URL(page.url());
  if (rightQueueURL.searchParams.get("source_eval_report_id") !== rightReportID) {
    throw new Error("right compare queue handoff missing source_eval_report_id");
  }
  if (rightQueueURL.searchParams.get("tenant_id") !== tenantID) {
    throw new Error("right compare queue handoff missing tenant_id");
  }
  if (rightQueueURL.searchParams.get("compare_origin_only") !== "true" || rightQueueURL.searchParams.get("status") !== "open") {
    throw new Error("right compare queue handoff missing canonical compare queue filters");
  }

  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(createOnlyTenantID) + "&left_report_id=" + encodeURIComponent(createOnlyLeftReportID) + "&right_report_id=" + encodeURIComponent(createOnlyRightReportID));
  await page.waitForSelector("text=Comparison summary");
  const createOnlyLeftText = (await page.textContent("#createLeftCaseButton")).trim();
  if (createOnlyLeftText !== "Create case from left") {
    throw new Error("left primary action should stay on create when no open compare follow-up exists");
  }
  await page.click("#createLeftCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const createdLeftURL = new URL(page.url());
  const leftCreatedCaseID = createdLeftURL.searchParams.get("case_id");
  if (!leftCreatedCaseID) {
    throw new Error("left compare-to-case handoff missing case_id");
  }
  if (createdLeftURL.searchParams.get("tenant_id") !== createOnlyTenantID) {
    throw new Error("left compare-to-case handoff missing tenant_id");
  }
  await assertCaseSource(page, baseURL, leftCreatedCaseID, createOnlyTenantID, createOnlyLeftReportID);

  await page.goto(baseURL + "/admin/eval-report-compare?tenant_id=" + encodeURIComponent(createOnlyTenantID) + "&left_report_id=" + encodeURIComponent(createOnlyLeftReportID) + "&right_report_id=" + encodeURIComponent(createOnlyRightReportID));
  await page.waitForSelector("text=Comparison summary");
  await page.click("#createRightCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const createdRightURL = new URL(page.url());
  const rightCreatedCaseID = createdRightURL.searchParams.get("case_id");
  if (!rightCreatedCaseID) {
    throw new Error("right compare-to-case handoff missing case_id");
  }
  if (createdRightURL.searchParams.get("tenant_id") !== createOnlyTenantID) {
    throw new Error("right compare-to-case handoff missing tenant_id");
  }
  await assertCaseSource(page, baseURL, rightCreatedCaseID, createOnlyTenantID, createOnlyRightReportID);

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

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-eval-compare-admin-smoke", leftReportID, rightReportID, leftCase.ID, rightCase.ID, "tenant-eval-compare-admin-create", createOnlyLeftReportID, createOnlyRightReportID)
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
	reportNoReuseID := materializeEvalRunReport(t, "tenant-eval-admin-smoke", evalsvc.RunStatusFailed, "secondary failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Follow-up No Reuse", "Source Follow-up No Reuse")
	comparePeerReportID := materializeEvalRunReport(t, "tenant-eval-admin-smoke", evalsvc.RunStatusSucceeded, "compare peer detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Peer", "Source Compare Peer")
	_ = materializeEvalRunReport(t, "tenant-eval-admin-smoke", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset No Follow-up", "Source No Follow-up")
	reportItem, err := reportService.GetEvalReport(context.Background(), reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if len(reportItem.BadCases) == 0 {
		t.Fatal("reportItem.BadCases is empty")
	}
	badCaseEvalCaseID := reportItem.BadCases[0].EvalCaseID
	reportNoReuseItem, err := reportService.GetEvalReport(context.Background(), reportNoReuseID)
	if err != nil {
		t.Fatalf("GetEvalReport(reportNoReuseID) error = %v", err)
	}
	if len(reportNoReuseItem.BadCases) == 0 {
		t.Fatal("reportNoReuseItem.BadCases is empty")
	}
	noReuseBadCaseEvalCaseID := reportNoReuseItem.BadCases[0].EvalCaseID

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
	openCompareFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-admin-smoke",
		Title:              "Open compare-origin follow-up",
		Summary:            "Open compare-origin summary",
		SourceEvalReportID: reportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  comparePeerReportID,
			RightEvalReportID: reportID,
			SelectedSide:      "right",
		},
		CreatedBy: "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateCase(openCompareFollowUp) error = %v", err)
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
	closedBadCaseFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-admin-smoke",
		Title:            "Closed bad-case follow-up",
		Summary:          "Closed bad-case follow-up summary",
		SourceEvalCaseID: badCaseEvalCaseID,
		CreatedBy:        "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateCase(closedBadCaseFollowUp) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedBadCaseFollowUp.ID, "operator-eval"); err != nil {
		t.Fatalf("CloseCase(closedBadCaseFollowUp) error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	openBadCaseFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-admin-smoke",
		Title:            "Open bad-case follow-up",
		Summary:          "Open bad-case follow-up summary",
		SourceEvalCaseID: badCaseEvalCaseID,
		CreatedBy:        "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateCase(openBadCaseFollowUp) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-admin-smoke",
		Title:            "Open bad-case follow-up without report-level case",
		Summary:          "Open bad-case follow-up summary without report-level case",
		SourceEvalCaseID: noReuseBadCaseEvalCaseID,
		CreatedBy:        "operator-eval",
	}); err != nil {
		t.Fatalf("CreateCase(noReuseBadCaseFollowUp) error = %v", err)
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
const badCaseEvalCaseID = process.argv[5];
const latestBadCaseFollowUpID = process.argv[6];
const reportNoReuseID = process.argv[7];
const latestCompareFollowUpID = process.argv[8];

async function assertCasePayload(page, apiBaseURL, caseID, tenantID, expectedReportID, expectedEvalCaseID) {
  await page.goto(apiBaseURL + "/api/v1/cases/" + encodeURIComponent(caseID) + "?tenant_id=" + encodeURIComponent(tenantID));
  await page.waitForSelector("body");
  const payload = JSON.parse(await page.textContent("body"));
  if (payload.source_eval_report_id !== expectedReportID) {
    throw new Error("unexpected source_eval_report_id for " + caseID + ": " + payload.source_eval_report_id);
  }
  if ((expectedEvalCaseID || "") !== (payload.source_eval_case_id || "")) {
    throw new Error("unexpected source_eval_case_id for " + caseID + ": " + payload.source_eval_case_id);
  }
}

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
  if (!followUpSummary.includes("1 compare-open")) {
    throw new Error("compare-origin follow-up count missing from list row: " + followUpSummary);
  }
  if (!followUpSummary.includes("latest open")) {
    throw new Error("latest follow-up case status missing from list row: " + followUpSummary);
  }
  const latestCaseHref = await page.getAttribute("#reportRows tr td:nth-child(5) a", "href");
  if (!latestCaseHref || !latestCaseHref.includes("/admin/cases?") || !latestCaseHref.includes("case_id=")) {
    throw new Error("latest case handoff link missing from list row");
  }
  const compareQueueHref = await page.getAttribute("#reportRows tr td:nth-child(5) a:nth-of-type(2)", "href");
  if (!compareQueueHref || !compareQueueHref.includes("/admin/cases?") || !compareQueueHref.includes("source_eval_report_id=" + encodeURIComponent(reportID)) || !compareQueueHref.includes("compare_origin_only=true") || !compareQueueHref.includes("status=open")) {
    throw new Error("compare queue handoff missing from list row");
  }
  const rowPrimaryActionText = (await page.textContent("#reportRows tr td:nth-child(7) a")).trim();
  if (rowPrimaryActionText !== "Open existing case") {
    throw new Error("row-level primary report action did not render backend-owned reuse label: " + rowPrimaryActionText);
  }
  const rowPrimaryActionHref = await page.getAttribute("#reportRows tr td:nth-child(7) a", "href");
  if (!rowPrimaryActionHref || !rowPrimaryActionHref.includes("/admin/cases?") || !rowPrimaryActionHref.includes("case_id=")) {
    throw new Error("row-level primary report action missing canonical case handoff");
  }
  const detailLatestCaseHref = await page.getAttribute("#openLatestCaseLink", "href");
  if (!detailLatestCaseHref || !detailLatestCaseHref.includes("/admin/cases?") || !detailLatestCaseHref.includes("case_id=")) {
    throw new Error("latest case handoff link missing from detail pane");
  }
  const detailLatestCompareHref = await page.getAttribute("#openLatestCompareCaseLink", "href");
  if (!detailLatestCompareHref || !detailLatestCompareHref.includes("case_id=" + encodeURIComponent(latestCompareFollowUpID))) {
    throw new Error("latest compare-origin case handoff missing from detail pane");
  }
  const detailCompareQueueHref = await page.getAttribute("#openCompareCasesLink", "href");
  if (!detailCompareQueueHref || !detailCompareQueueHref.includes("/admin/cases?") || !detailCompareQueueHref.includes("source_eval_report_id=" + encodeURIComponent(reportID)) || !detailCompareQueueHref.includes("compare_origin_only=true") || !detailCompareQueueHref.includes("status=open")) {
    throw new Error("compare-origin queue handoff missing from detail pane");
  }
  const compareFollowUpSummary = (await page.textContent("#reportDetail")).trim();
  if (!compareFollowUpSummary.includes("Compare-origin follow-up") || !compareFollowUpSummary.includes("1 cases / 1 open")) {
    throw new Error("compare-origin summary missing from detail pane");
  }
  const primaryCaseAction = (await page.textContent("#createCaseButton")).trim();
  if (primaryCaseAction !== "Open existing case") {
    throw new Error("primary report case action did not switch to existing case reuse: " + primaryCaseAction);
  }
  const primaryCaseActionMode = await page.getAttribute("#createCaseButton", "data-action");
  if (primaryCaseActionMode !== "open-existing") {
    throw new Error("primary report case action mode missing reuse state: " + primaryCaseActionMode);
  }
  const primaryCaseTargetHref = await page.getAttribute("#createCaseButton", "data-target-href");
  if (!primaryCaseTargetHref || !primaryCaseTargetHref.includes("case_id=" + encodeURIComponent(openFollowUp.ID))) {
    throw new Error("primary report case action target missing reused case handoff");
  }
  const existingBadCaseActionHref = await page.locator("#reportDetail .bad-case-item a").evaluateAll((elements) => {
    const match = elements.find((element) => element.textContent && element.textContent.includes("Open existing bad-case case"));
    return match ? match.getAttribute("href") : "";
  });
  if (!existingBadCaseActionHref || !existingBadCaseActionHref.includes("case_id=" + encodeURIComponent(openBadCaseFollowUp.ID))) {
    throw new Error("existing bad-case reuse handoff missing from detail pane");
  }
  await page.click("#quickViewAllReports");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("needs_follow_up") === null && new URL(window.location.href).searchParams.get("bad_case_needs_follow_up") === null);
  await page.click("#quickViewBadCaseNeedsFollowUp");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("bad_case_needs_follow_up") === "true");
  const unresolvedVisibleCount = (await page.textContent("#visibleCount")).trim();
  if (unresolvedVisibleCount !== "1") {
    throw new Error("unexpected visibleCount after unresolved bad-case quick view: " + unresolvedVisibleCount);
  }
  const badCaseFollowUpFilterValue = await page.$eval("#bad_case_needs_follow_up", (el) => el.value);
  if (badCaseFollowUpFilterValue !== "true") {
    throw new Error("bad_case_needs_follow_up filter was not synced to quick view");
  }
  const unresolvedSummary = (await page.textContent("#reportRows tr td:nth-child(5)")).trim();
  if (!unresolvedSummary.includes("1 bad cases uncovered")) {
    throw new Error("unresolved bad-case summary missing from list row: " + unresolvedSummary);
  }
  const unresolvedLinkHref = await page.locator("#reportRows tr td:nth-child(5) a").evaluateAll((elements) => {
    const match = elements.find((element) => element.textContent && element.textContent.includes("Open unresolved bad cases"));
    return match ? match.getAttribute("href") : "";
  });
  if (!unresolvedLinkHref || !unresolvedLinkHref.includes("/admin/eval-reports?") || !unresolvedLinkHref.includes("bad_case_needs_follow_up=true") || !unresolvedLinkHref.includes("report_id=" + encodeURIComponent(reportNoReuseID)) || !unresolvedLinkHref.includes("selected_report_id=" + encodeURIComponent(reportNoReuseID))) {
    throw new Error("row-level unresolved bad-case handoff missing from eval report row");
  }
  await page.click("#quickViewAllReports");
  await page.waitForFunction(() => {
    const params = new URL(window.location.href).searchParams;
    return params.get("needs_follow_up") === null && params.get("bad_case_needs_follow_up") === null;
  });
  await page.waitForFunction(() => document.querySelector("#visibleCount") && document.querySelector("#visibleCount").textContent.trim() === "2");
  await page.waitForFunction((expectedReportID) => {
    const params = new URL(window.location.href).searchParams;
    return params.get("selected_report_id") === expectedReportID && params.get("report_id") === null;
  }, reportID);
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
  const latestCaseHrefBeforeCreate = await page.getAttribute("#openLatestCaseLink", "href");
  if (!latestCaseHrefBeforeCreate || !latestCaseHrefBeforeCreate.includes("case_id=")) {
    throw new Error("latest case handoff missing before create");
  }
  const expectedExistingCaseID = new URL("http://local" + latestCaseHrefBeforeCreate).searchParams.get("case_id");
  if (!expectedExistingCaseID) {
    throw new Error("unable to parse existing case_id from latest case handoff");
  }
  const urlAfterLoad = new URL(page.url());
  if (urlAfterLoad.searchParams.get("selected_report_id") !== reportID || urlAfterLoad.searchParams.get("report_id") !== null) {
    throw new Error("selected_report_id not synced into URL");
  }
  await page.click("#createCaseButton");
  await page.waitForURL(/\/admin\/cases\?/);
  const createdCaseURL = new URL(page.url());
  const createdCaseID = createdCaseURL.searchParams.get("case_id");
  if (!createdCaseID) {
    throw new Error("create-case handoff missing case_id");
  }
  if (createdCaseID !== expectedExistingCaseID) {
    throw new Error("create-case did not reuse existing open follow-up case");
  }
  if (createdCaseURL.searchParams.get("tenant_id") !== tenantID) {
    throw new Error("create-case handoff missing tenant_id");
  }
  await page.waitForSelector("text=Source eval report summary");
  const evalReportLink = await page.getAttribute("#openEvalReportLink", "href");
  if (!evalReportLink || !evalReportLink.includes(reportID)) {
    throw new Error("created case did not preserve source_eval_report_id");
  }
  await assertCasePayload(page, baseURL, createdCaseID, tenantID, reportID, "");

  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&selected_report_id=" + encodeURIComponent(reportID));
  await page.waitForSelector("text=Bad cases");
  const badCaseButton = await page.locator("[data-create-bad-case]").first();
  const sourceEvalCaseID = await badCaseButton.getAttribute("data-create-bad-case");
  if (!sourceEvalCaseID) {
    throw new Error("missing bad-case eval_case_id in detail action");
  }
  if (sourceEvalCaseID !== badCaseEvalCaseID) {
    throw new Error("unexpected bad-case eval_case_id: " + sourceEvalCaseID);
  }
  const badCaseSummaryText = await page.locator(".bad-case-item").first().textContent();
  if (!badCaseSummaryText.includes("2 cases / 1 open")) {
    throw new Error("bad-case follow-up summary missing from detail: " + badCaseSummaryText);
  }
  const latestBadCaseHref = await page.locator("text=Open latest bad-case case").first().getAttribute("href");
  if (!latestBadCaseHref || !latestBadCaseHref.includes("case_id=" + encodeURIComponent(latestBadCaseFollowUpID))) {
    throw new Error("bad-case latest case handoff missing canonical case_id");
  }
  const badCaseSliceHref = await page.locator("text=Open bad-case follow-up slice").first().getAttribute("href");
  if (!badCaseSliceHref || !badCaseSliceHref.includes("source_eval_case_id=" + encodeURIComponent(sourceEvalCaseID))) {
    throw new Error("bad-case follow-up slice handoff missing source_eval_case_id");
  }
  await page.click("#badCaseQuickViewNeedsFollowUp");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("detail_bad_case_needs_follow_up") === "true");
  const badCaseCountWithFollowUp = await page.locator(".bad-case-item").count();
  if (badCaseCountWithFollowUp !== 1) {
    throw new Error("unexpected bad-case count after needs-follow-up filter: " + badCaseCountWithFollowUp);
  }
  await page.click("#badCaseQuickViewNoFollowUp");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("detail_bad_case_needs_follow_up") === "false");
  await page.waitForSelector("text=No bad cases were materialized for this eval report.");
  await page.click("#quickViewBadCaseNeedsFollowUp");
  await page.waitForFunction((expectedReportID) => {
    const params = new URL(window.location.href).searchParams;
    return params.get("bad_case_needs_follow_up") === "true" &&
      params.get("detail_bad_case_needs_follow_up") === null &&
      params.get("selected_report_id") === expectedReportID &&
      params.get("report_id") === null;
  }, reportNoReuseID);
  const unresolvedBadCaseCount = await page.locator(".bad-case-item").count();
  if (unresolvedBadCaseCount !== 1) {
    throw new Error("detail did not reset to canonical unresolved bad-case slice after list quick view: " + unresolvedBadCaseCount);
  }
  const unresolvedDetailText = await page.locator(".bad-case-item").first().textContent();
  if (!unresolvedDetailText.includes("0 open")) {
    throw new Error("expected unresolved bad-case detail after list quick view reset: " + unresolvedDetailText);
  }
  await page.click("#quickViewAllReports");
  await page.waitForFunction(() => {
    const params = new URL(window.location.href).searchParams;
    return params.get("bad_case_needs_follow_up") === null && params.get("detail_bad_case_needs_follow_up") === null;
  });
  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&selected_report_id=" + encodeURIComponent(reportID));
  await page.waitForSelector("text=Bad cases");
  await page.click("#badCaseQuickViewAll");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("detail_bad_case_needs_follow_up") === null);
  await page.waitForSelector("[data-create-bad-case]");
  const restoredBadCaseCount = await page.locator(".bad-case-item").count();
  if (restoredBadCaseCount !== 1) {
    throw new Error("unexpected bad-case count after clearing filter: " + restoredBadCaseCount);
  }
  const filteredBadCaseButton = await page.locator("[data-create-bad-case]").first();
  if ((await filteredBadCaseButton.getAttribute("data-create-bad-case")) !== sourceEvalCaseID) {
    throw new Error("bad-case action did not restore the expected eval_case_id after clearing filter");
  }
  await page.click("#badCaseQuickViewNoFollowUp");
  await page.waitForFunction(() => new URL(window.location.href).searchParams.get("detail_bad_case_needs_follow_up") === "false");
  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&selected_report_id=" + encodeURIComponent(reportNoReuseID));
  await page.waitForSelector("text=Bad cases");
  await page.click("#badCaseQuickViewNoFollowUp");
  await page.waitForFunction(() => {
    const params = new URL(window.location.href).searchParams;
    return params.get("selected_report_id") === reportNoReuseID && params.get("report_id") === null && params.get("detail_bad_case_needs_follow_up") === "false";
  });
  const filteredReportCaseButton = page.locator("#createCaseButton");
  await filteredReportCaseButton.click();
  await page.waitForURL(/\/admin\/cases\?/);
  const filteredReportCaseURL = new URL(page.url());
  const filteredReportCaseID = filteredReportCaseURL.searchParams.get("case_id");
  if (!filteredReportCaseID) {
    throw new Error("filtered report create-case handoff missing case_id");
  }
  await page.goto(baseURL + "/api/v1/cases/" + encodeURIComponent(filteredReportCaseID) + "?tenant_id=" + encodeURIComponent(tenantID));
  await page.waitForSelector("body");
  const filteredReportCasePayload = JSON.parse(await page.textContent("body"));
  if (filteredReportCasePayload.source_eval_report_id !== reportNoReuseID) {
    throw new Error("filtered report create-case used the wrong report lineage: " + filteredReportCasePayload.source_eval_report_id);
  }
  if (!String(filteredReportCasePayload.summary || "").includes("Bad cases: 1")) {
    throw new Error("filtered report create-case summary lost canonical bad_case_count: " + filteredReportCasePayload.summary);
  }

  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&selected_report_id=" + encodeURIComponent(reportID));
  await page.waitForSelector("text=Bad cases");
  await page.waitForSelector("[data-create-bad-case]");
  const finalBadCaseButton = await page.locator("[data-create-bad-case]").first();
  await finalBadCaseButton.click();
  await page.waitForURL(/\/admin\/cases\?/);
  const createdBadCaseURL = new URL(page.url());
  const createdBadCaseID = createdBadCaseURL.searchParams.get("case_id");
  if (!createdBadCaseID) {
    throw new Error("bad-case create-case handoff missing case_id");
  }
  await assertCasePayload(page, baseURL, createdBadCaseID, tenantID, reportID, sourceEvalCaseID);

  await page.goto(baseURL + "/admin/eval-reports?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&report_id=missing-report&selected_report_id=missing-report");
  await page.waitForSelector("text=No durable eval reports matched the current slice.");
  const failedURL = new URL(page.url());
  if (failedURL.searchParams.get("report_id") !== "missing-report") {
    throw new Error("canonical report_id filter was unexpectedly cleared after missing-report handoff");
  }
  if (failedURL.searchParams.get("selected_report_id")) {
    throw new Error("stale selected_report_id remained in URL after missing-report handoff");
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

	cmd := exec.Command("node", scriptPath, server.URL, "tenant-eval-admin-smoke", reportID, badCaseEvalCaseID, openBadCaseFollowUp.ID, reportNoReuseID, openCompareFollowUp.ID)
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
	if !strings.Contains(body, "Open eval API detail") {
		t.Fatal("source eval case handoff missing from cases page HTML")
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
	if !strings.Contains(body, "Assign to me") {
		t.Fatal("row-level assign-to-me action missing from cases page HTML")
	}
	if !strings.Contains(body, "Return to queue") {
		t.Fatal("row-level return-to-queue action missing from cases page HTML")
	}
	if !strings.Contains(body, "Close from queue") {
		t.Fatal("row-level close-from-queue action missing from cases page HTML")
	}
	if !strings.Contains(body, "Reopen from queue") {
		t.Fatal("row-level reopen-from-queue action missing from cases page HTML")
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
	if !strings.Contains(body, "Source eval case") {
		t.Fatal("source eval case detail missing from cases page HTML")
	}
	if !strings.Contains(body, "Run-backed cases") {
		t.Fatal("run-backed-cases quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Source eval run") {
		t.Fatal("source eval run detail missing from cases page HTML")
	}
	if !strings.Contains(body, "Open eval run") {
		t.Fatal("eval run handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Open eval run API detail") {
		t.Fatal("eval run api handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "Open cases") {
		t.Fatal("open-cases quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Eval-backed cases") {
		t.Fatal("eval-backed quick view missing from cases page HTML")
	}
	if !strings.Contains(body, "Compare follow-ups") {
		t.Fatal("compare-follow-ups quick view missing from cases page HTML")
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
	if !strings.Contains(body, "Compare origin") {
		t.Fatal("compare origin section missing from cases page HTML")
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
	if !strings.Contains(body, "Open compare origin") {
		t.Fatal("compare origin handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "data-case-compare-link") {
		t.Fatal("row-level compare handoff missing from cases page HTML")
	}
	if !strings.Contains(body, "<option value=\"closed\">Closed</option>") {
		t.Fatal("closed status filter missing from cases page HTML")
	}
}

func TestAdminCasesPageRuntimeSmoke(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-case-admin-smoke", evalsvc.RunStatusFailed, "failure detail")
	compareRightReportID := "eval-report-compare-other"
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
		SourceEvalCaseID:   reportItem.BadCases[0].EvalCaseID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  reportID,
			RightEvalReportID: compareRightReportID,
			SelectedSide:      "left",
		},
		CreatedBy: "operator-case",
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
	runBackedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:         "tenant-case-admin-smoke",
		Title:            "Investigate failed eval run",
		Summary:          "Follow up a durable eval run directly from cases",
		SourceEvalRunID:  reportItem.RunID,
		SourceEvalCaseID: reportItem.BadCases[0].EvalCaseID,
		CreatedBy:        "operator-case",
	})
	if err != nil {
		t.Fatalf("CreateCase(runBacked) error = %v", err)
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
const compareRightReportID = process.argv[11];
const runBackedCaseID = process.argv[12];
const evalRunID = process.argv[13];

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
  if (!detailText.includes("Selected eval case")) throw new Error("missing selected eval case summary");
  const caseDetailText = await page.textContent("#caseDetail");
  if (!caseDetailText.includes("Compare origin")) throw new Error("missing compare origin section");
  if (!caseDetailText.includes("Source eval case")) throw new Error("missing source eval case section");
  if (!caseDetailText.includes(reportID)) throw new Error("missing compare left report id");
  if (!caseDetailText.includes(compareRightReportID)) throw new Error("missing compare right report id");
  const evalCaseHref = await page.getAttribute("#openEvalCaseAPILink", "href");
  if (!evalCaseHref || !evalCaseHref.includes("/api/v1/eval-cases/")) {
    throw new Error("source eval case api handoff missing selected eval case");
  }
  const evalLaneHref = await page.getAttribute("#openEvalReportsLink", "href");
  if (!evalLaneHref || !evalLaneHref.includes("report_id=" + encodeURIComponent(reportID))) {
    throw new Error("eval report lane handoff missing report_id");
  }
  const compareHref = await page.getAttribute("#openEvalCompareLink", "href");
  if (!compareHref || !compareHref.includes("left_report_id=" + encodeURIComponent(reportID)) || !compareHref.includes("right_report_id=" + encodeURIComponent(compareRightReportID))) {
    throw new Error("compare origin handoff drifted");
  }
  const rowCompareHref = await page.getAttribute('[data-case-compare-link="' + linkedCaseID + '"]', "href");
  if (!rowCompareHref || !rowCompareHref.includes("left_report_id=" + encodeURIComponent(reportID)) || !rowCompareHref.includes("right_report_id=" + encodeURIComponent(compareRightReportID))) {
    throw new Error("row-level compare handoff drifted");
  }
  await page.fill("#caseActor", "queue-owner");
  await page.click('[data-case-assign-id="' + linkedCaseID + '"]');
  await page.waitForFunction(
    (caseID) => {
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return row && row.textContent && row.textContent.includes("queue-owner");
    },
    linkedCaseID
  );
  const assignedDetail = await page.textContent("#caseDetail");
  if (!assignedDetail.includes("queue-owner")) throw new Error("row-level assign did not refresh detail ownership");
  await page.click("#compareFollowUpsQuickView");
  await page.waitForFunction(() => document.querySelector("#visibleCount")?.textContent?.trim() === "1");
  const compareVisibleCount = await page.textContent("#visibleCount");
  if (compareVisibleCount.trim() !== "1") throw new Error("compare quick view did not narrow to compare-derived cases");
  const compareURL = new URL(page.url());
  if (compareURL.searchParams.get("compare_origin_only") !== "true") {
    throw new Error("compare_origin_only filter was not synced to quick view");
  }
  const compareOnlyStatus = await page.$eval("#status", (node) => node.value);
  if (compareOnlyStatus !== "open") throw new Error("compare quick view drifted open status");
  await page.click('[data-case-unassign-id="' + linkedCaseID + '"]');
  await page.waitForFunction(
    (caseID) => {
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return row && row.textContent && row.textContent.includes("Unassigned");
    },
    linkedCaseID
  );
  const unassignedDetail = await page.textContent("#caseDetail");
  if (!unassignedDetail.includes("currently unassigned")) throw new Error("row-level unassign did not refresh detail state");
  await page.click('[data-case-assign-id="' + linkedCaseID + '"]');
  await page.waitForFunction(
    (caseID) => {
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return row && row.textContent && row.textContent.includes("queue-owner");
    },
    linkedCaseID
  );
  await page.click('[data-case-close-id="' + linkedCaseID + '"]');
  await page.waitForFunction(() => document.querySelector("#visibleCount")?.textContent?.trim() === "0");
  const closedVisibleCount = await page.textContent("#visibleCount");
  if (closedVisibleCount.trim() !== "0") throw new Error("row-level close did not remove case from open compare queue");
  const queueText = await page.textContent("#caseListState");
  if (!queueText.includes("No cases matched the current slice.")) throw new Error("closed compare queue did not enter empty state");
  await page.goto(baseURL + "/admin/cases?tenant_id=" + encodeURIComponent(tenantID) + "&status=closed&compare_origin_only=true&limit=10&case_id=" + encodeURIComponent(linkedCaseID));
  await page.waitForFunction(
    (caseID) => {
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return row && row.textContent && row.textContent.includes("closed");
    },
    linkedCaseID
  );
  await page.click('[data-case-reopen-id="' + linkedCaseID + '"]');
  await page.waitForFunction(
    (caseID) => {
      const statusFilter = document.querySelector("#status");
      const compareFilter = document.querySelector("#compare_origin_only");
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return statusFilter && statusFilter.value === "open" && compareFilter && compareFilter.value === "true" && row && row.textContent && row.textContent.includes("open");
    },
    linkedCaseID
  );
  const reopenedVisibleCount = await page.textContent("#visibleCount");
  if (reopenedVisibleCount.trim() !== "1") throw new Error("row-level reopen did not return case to the open queue");
  const reopenedURL = new URL(page.url());
  if (reopenedURL.searchParams.get("compare_origin_only") !== "true") throw new Error("row-level reopen drifted compare queue filter");
  const reopenedDetail = await page.textContent("#caseDetail");
  if (!reopenedDetail.includes("queue-owner")) throw new Error("row-level reopen did not preserve case ownership in detail");

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
  await page.goto(baseURL + "/admin/cases?tenant_id=" + encodeURIComponent(tenantID) + "&limit=10&case_id=" + encodeURIComponent(runBackedCaseID));
  await page.waitForFunction(
    (caseID) => {
      const row = document.querySelector('[data-case-row="' + caseID + '"]');
      return row && row.textContent && row.textContent.includes("Eval run-backed");
    },
    runBackedCaseID
  );
  const runDetailText = await page.textContent("#caseDetail");
  if (!runDetailText.includes("Source eval run")) throw new Error("missing source eval run detail section");
  if (!runDetailText.includes(evalRunID)) throw new Error("missing source eval run id in detail");
  const evalRunHref = await page.getAttribute("#openEvalRunLink", "href");
  if (!evalRunHref || !evalRunHref.includes("run_id=" + encodeURIComponent(evalRunID))) {
    throw new Error("eval run handoff drifted");
  }
  const evalRunAPIHref = await page.getAttribute("#openEvalRunAPILink", "href");
  if (!evalRunAPIHref || !evalRunAPIHref.includes("/api/v1/eval-runs/" + encodeURIComponent(evalRunID))) {
    throw new Error("eval run api handoff drifted");
  }
  await page.click("#runBackedCasesQuickView");
  await page.waitForFunction(() => document.querySelector("#visibleCount")?.textContent?.trim() === "1");
  const runURL = new URL(page.url());
  if (runURL.searchParams.get("run_backed_only") !== "true") {
    throw new Error("run-backed quick view did not sync run_backed_only");
  }
  const runVisibleCount = await page.textContent("#visibleCount");
  if (runVisibleCount.trim() !== "1") throw new Error("run-backed quick view did not narrow to run-backed cases");
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
		compareRightReportID,
		runBackedCase.ID,
		reportItem.RunID,
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
