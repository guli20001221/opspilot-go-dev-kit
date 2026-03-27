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

func TestCompareEvalReportsReturnsTypedSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-eval-report-compare", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare A", "Source Left")
	rightReportID := materializeEvalRunReport(t, "tenant-eval-report-compare", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare B", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-eval-report-compare&left_report_id=" + leftReportID + "&right_report_id=" + rightReportID)
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
	var raw map[string]any
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		t.Fatalf("Unmarshal(raw) error = %v", err)
	}
	var got struct {
		Left struct {
			ReportID        string  `json:"report_id"`
			TenantID        string  `json:"tenant_id"`
			RunID           string  `json:"run_id"`
			DatasetID       string  `json:"dataset_id"`
			DatasetName     string  `json:"dataset_name"`
			RunStatus       string  `json:"run_status"`
			Status          string  `json:"status"`
			Summary         string  `json:"summary"`
			TotalItems      int     `json:"total_items"`
			RecordedResults int     `json:"recorded_results"`
			PassedItems     int     `json:"passed_items"`
			FailedItems     int     `json:"failed_items"`
			MissingResults  int     `json:"missing_results"`
			AverageScore    float64 `json:"average_score"`
			JudgeVersion    string  `json:"judge_version"`
			VersionID       string  `json:"version_id"`
			BadCaseCount    int     `json:"bad_case_count"`
		} `json:"left"`
		Right struct {
			ReportID        string  `json:"report_id"`
			TenantID        string  `json:"tenant_id"`
			RunID           string  `json:"run_id"`
			DatasetID       string  `json:"dataset_id"`
			DatasetName     string  `json:"dataset_name"`
			RunStatus       string  `json:"run_status"`
			Status          string  `json:"status"`
			Summary         string  `json:"summary"`
			TotalItems      int     `json:"total_items"`
			RecordedResults int     `json:"recorded_results"`
			PassedItems     int     `json:"passed_items"`
			FailedItems     int     `json:"failed_items"`
			MissingResults  int     `json:"missing_results"`
			AverageScore    float64 `json:"average_score"`
			JudgeVersion    string  `json:"judge_version"`
			VersionID       string  `json:"version_id"`
			BadCaseCount    int     `json:"bad_case_count"`
		} `json:"right"`
		Summary struct {
			SameTenant          bool    `json:"same_tenant"`
			SameDataset         bool    `json:"same_dataset"`
			SameRunStatus       bool    `json:"same_run_status"`
			JudgeVersionChanged bool    `json:"judge_version_changed"`
			MetadataChanged     bool    `json:"metadata_changed"`
			FailedItemsDelta    int     `json:"failed_items_delta"`
			AverageScoreDelta   float64 `json:"average_score_delta"`
			BadCaseOverlapCount int     `json:"bad_case_overlap_count"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(bodyBytes, &got); err != nil {
		t.Fatalf("Unmarshal(got) error = %v", err)
	}
	leftRaw, ok := raw["left"].(map[string]any)
	if !ok {
		t.Fatalf("raw left payload = %#v, want object", raw["left"])
	}
	rightRaw, ok := raw["right"].(map[string]any)
	if !ok {
		t.Fatalf("raw right payload = %#v, want object", raw["right"])
	}
	if _, ok := leftRaw["metadata"]; ok {
		t.Fatalf("left payload unexpectedly includes metadata: %#v", leftRaw)
	}
	if _, ok := leftRaw["bad_cases"]; ok {
		t.Fatalf("left payload unexpectedly includes bad_cases: %#v", leftRaw)
	}
	if _, ok := rightRaw["metadata"]; ok {
		t.Fatalf("right payload unexpectedly includes metadata: %#v", rightRaw)
	}
	if _, ok := rightRaw["bad_cases"]; ok {
		t.Fatalf("right payload unexpectedly includes bad_cases: %#v", rightRaw)
	}
	if got.Left.ReportID != leftReportID || got.Right.ReportID != rightReportID {
		t.Fatalf("left/right ids = %#v, want %q and %q", got, leftReportID, rightReportID)
	}
	if got.Left.BadCaseCount != 0 || got.Right.BadCaseCount != 1 {
		t.Fatalf("BadCaseCount = left:%d right:%d, want left=0 right=1", got.Left.BadCaseCount, got.Right.BadCaseCount)
	}
	if !got.Summary.SameTenant {
		t.Fatal("SameTenant = false, want true")
	}
	if got.Summary.SameDataset {
		t.Fatal("SameDataset = true, want false")
	}
	if got.Summary.SameRunStatus {
		t.Fatal("SameRunStatus = true, want false")
	}
	if got.Summary.JudgeVersionChanged {
		t.Fatalf("JudgeVersionChanged = true, want false for placeholder fixtures")
	}
	if !got.Summary.MetadataChanged {
		t.Fatalf("MetadataChanged = false, want true")
	}
	if got.Summary.FailedItemsDelta <= 0 {
		t.Fatalf("FailedItemsDelta = %d, want positive", got.Summary.FailedItemsDelta)
	}
	if got.Summary.AverageScoreDelta >= 0 {
		t.Fatalf("AverageScoreDelta = %v, want negative", got.Summary.AverageScoreDelta)
	}
	if got.Summary.BadCaseOverlapCount != 0 {
		t.Fatalf("BadCaseOverlapCount = %d, want 0 for disjoint fixtures", got.Summary.BadCaseOverlapCount)
	}
}

func TestCompareEvalReportsRejectsMissingTenantID(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?left_report_id=" + reportID + "&right_report_id=" + reportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCompareEvalReportsRejectsMissingLeftReportID(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-eval-report-http&right_report_id=" + reportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCompareEvalReportsRejectsMissingRightReportID(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-eval-report-http&left_report_id=" + reportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCompareEvalReportsRejectsWrongTenant(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-left", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare A", "Source Left")
	rightReportID := materializeEvalRunReport(t, "tenant-right", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare B", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-left&left_report_id=" + leftReportID + "&right_report_id=" + rightReportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestCompareEvalReportsRejectsMissingReport(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-eval-report-http&left_report_id=" + reportID + "&right_report_id=missing-report")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
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

func TestGetEvalReportRejectsMissingTenantID(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetEvalReportRejectsWrongTenant(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusSucceeded, "success detail")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-other")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestListEvalReportsRejectsInvalidStatus(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval&status=queued")
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
