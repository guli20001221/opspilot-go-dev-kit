package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
)

func TestGetEvalReportReturnsMaterializedDetail(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusFailed, "failure detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-http")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got evalReportResponse
	if err := json.Unmarshal(bodyBytes, &got); err != nil {
		t.Fatalf("Unmarshal(got) error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		t.Fatalf("Unmarshal(raw) error = %v", err)
	}
	if got.ReportID != reportID {
		t.Fatalf("ReportID = %q, want %q", got.ReportID, reportID)
	}
	if got.RunStatus != evalsvc.RunStatusFailed {
		t.Fatalf("RunStatus = %q, want %q", got.RunStatus, evalsvc.RunStatusFailed)
	}
	if len(got.BadCases) == 0 {
		t.Fatal("BadCases is empty")
	}
	if len(got.Metadata) == 0 {
		t.Fatal("Metadata is empty")
	}
	if _, ok := raw["bad_cases"]; !ok {
		t.Fatalf("detail response missing bad_cases: %#v", raw)
	}
	if _, ok := raw["metadata"]; !ok {
		t.Fatalf("detail response missing metadata: %#v", raw)
	}
}

func TestListEvalReportsSupportsFiltersAndPagination(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	_ = materializeEvalRunReport(t, "tenant-eval-report-list", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset One", "Source A")
	_ = materializeEvalRunReport(t, "tenant-eval-report-list", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Two", "Source B")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-list&status=ready&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var page listEvalReportsResponse
	if err := json.Unmarshal(bodyBytes, &page); err != nil {
		t.Fatalf("Unmarshal(page) error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		t.Fatalf("Unmarshal(raw) error = %v", err)
	}
	if len(page.Reports) != 1 {
		t.Fatalf("len(page.Reports) = %d, want 1", len(page.Reports))
	}
	if page.Reports[0].RunStatus != evalsvc.RunStatusFailed {
		t.Fatalf("RunStatus = %q, want %q for newest row", page.Reports[0].RunStatus, evalsvc.RunStatusFailed)
	}
	if !page.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if page.NextOffset == nil || *page.NextOffset != 1 {
		t.Fatalf("NextOffset = %#v, want 1", page.NextOffset)
	}
	rawReports, ok := raw["reports"].([]any)
	if !ok || len(rawReports) != 1 {
		t.Fatalf("raw reports = %#v, want one item", raw["reports"])
	}
	rawItem, ok := rawReports[0].(map[string]any)
	if !ok {
		t.Fatalf("raw item = %#v, want object", rawReports[0])
	}
	if _, ok := rawItem["bad_cases"]; ok {
		t.Fatalf("list response unexpectedly included bad_cases: %#v", rawItem)
	}
	if _, ok := rawItem["metadata"]; ok {
		t.Fatalf("list response unexpectedly included metadata: %#v", rawItem)
	}
}

func TestListEvalReportsRejectsInvalidRunStatus(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval&run_status=queued")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func buildEvalReportFixture(t *testing.T, tenantID string, runStatus string, detail string) (*evalsvc.EvalReportService, string) {
	t.Helper()

	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, tenantID, runStatus, detail, caseService, evalCaseService, datasetService, runService, reportService, "Dataset One", "Source A")
	return reportService, reportID
}

func materializeEvalRunReport(t *testing.T, tenantID string, runStatus string, detail string, caseService *casesvc.Service, evalCaseService *evalsvc.Service, datasetService *evalsvc.DatasetService, runService *evalsvc.RunService, reportService *evalsvc.EvalReportService, datasetName string, caseTitle string) string {
	t.Helper()

	ctx := context.Background()
	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: tenantID,
		Title:    caseTitle,
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     tenantID,
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    tenantID,
		Name:        datasetName,
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: tenantID}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}
	run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  tenantID,
		DatasetID: dataset.ID,
		CreatedBy: "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	claimed, err := runService.ClaimQueuedRuns(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) == 0 {
		t.Fatal("ClaimQueuedRuns() returned no runs")
	}
	switch runStatus {
	case evalsvc.RunStatusSucceeded:
		if _, err := runService.MarkRunSucceeded(ctx, run.ID); err != nil {
			t.Fatalf("MarkRunSucceeded() error = %v", err)
		}
	case evalsvc.RunStatusFailed:
		if _, err := runService.MarkRunFailed(ctx, run.ID, detail); err != nil {
			t.Fatalf("MarkRunFailed() error = %v", err)
		}
	default:
		t.Fatalf("unsupported runStatus %q", runStatus)
	}
	item, err := reportService.MaterializeRunReport(ctx, run.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport() error = %v", err)
	}
	return item.ID
}
