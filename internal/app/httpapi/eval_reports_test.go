package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	if got.PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "create")
	}
	if got.PreferredFollowUpAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredFollowUpAction.SourceEvalReportID, reportID)
	}
	if len(got.BadCases) == 0 {
		t.Fatal("BadCases is empty")
	}
	if got.BadCases[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("BadCases[0].PreferredFollowUpAction.Mode = %q, want %q", got.BadCases[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.BadCases[0].PreferredFollowUpAction.SourceEvalCaseID != got.BadCases[0].EvalCaseID {
		t.Fatalf("BadCases[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", got.BadCases[0].PreferredFollowUpAction.SourceEvalCaseID, got.BadCases[0].EvalCaseID)
	}
	if got.PreferredCompareFollowUpAction.Mode != "none" {
		t.Fatalf("PreferredCompareFollowUpAction.Mode = %q, want %q", got.PreferredCompareFollowUpAction.Mode, "none")
	}
	if got.PreferredCompareFollowUpAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredCompareFollowUpAction.SourceEvalReportID, reportID)
	}
	if got.LinkedCaseSummary == nil {
		t.Fatal("LinkedCaseSummary is nil")
	}
	if got.LinkedCaseSummary.TotalCaseCount != 0 || got.LinkedCaseSummary.OpenCaseCount != 0 {
		t.Fatalf("LinkedCaseSummary = %#v, want zero-value summary", got.LinkedCaseSummary)
	}
	if got.BadCaseCount != 1 {
		t.Fatalf("BadCaseCount = %d, want 1", got.BadCaseCount)
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
	if _, ok := raw["bad_case_count"]; !ok {
		t.Fatalf("detail response missing bad_case_count: %#v", raw)
	}
}

func TestGetEvalReportIncludesFollowUpCaseSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-detail-followup", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Followup Detail", "Source Detail")

	closedCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-detail-followup",
		Title:              "Closed follow-up",
		Summary:            "closed summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedCase.ID, "operator-followup"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	openCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-detail-followup",
		Title:              "Open follow-up",
		Summary:            "open summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), openCase, "operator-followup"); err != nil {
		t.Fatalf("AssignCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-detail-followup")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseCount = %d, want 2", got.FollowUpCaseCount)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 1", got.OpenFollowUpCaseCount)
	}
	if got.LatestFollowUpCaseID == "" {
		t.Fatal("LatestFollowUpCaseID is empty")
	}
	if got.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", got.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if got.PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if got.PreferredFollowUpAction.CaseID != openCase.ID {
		t.Fatalf("PreferredFollowUpAction.CaseID = %q, want %q", got.PreferredFollowUpAction.CaseID, openCase.ID)
	}
	if got.PreferredFollowUpAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredFollowUpAction.SourceEvalReportID, reportID)
	}
	if got.LinkedCaseSummary == nil {
		t.Fatal("LinkedCaseSummary is nil")
	}
	if got.LinkedCaseSummary.TotalCaseCount != 2 {
		t.Fatalf("LinkedCaseSummary.TotalCaseCount = %d, want 2", got.LinkedCaseSummary.TotalCaseCount)
	}
	if got.LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.OpenCaseCount = %d, want 1", got.LinkedCaseSummary.OpenCaseCount)
	}
	if got.LinkedCaseSummary.LatestCaseID != openCase.ID {
		t.Fatalf("LinkedCaseSummary.LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, openCase.ID)
	}
	if got.LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.LinkedCaseSummary.LatestAssignedTo != "operator-followup" {
		t.Fatalf("LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "operator-followup")
	}
	if got.PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredLinkedCaseAction.CaseID != openCase.ID {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want %q", got.PreferredLinkedCaseAction.CaseID, openCase.ID)
	}
	if got.PreferredLinkedCaseAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredLinkedCaseAction.SourceEvalReportID = %q, want %q", got.PreferredLinkedCaseAction.SourceEvalReportID, reportID)
	}
}

func TestGetEvalReportIncludesBadCaseFollowUpCaseSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-badcase-followup", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Bad Case Followup", "Source Bad Case Followup")

	report, err := reportService.GetEvalReport(context.Background(), reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if len(report.BadCases) != 1 {
		t.Fatalf("len(report.BadCases) = %d, want 1", len(report.BadCases))
	}
	evalCaseID := report.BadCases[0].EvalCaseID

	closedCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-report-badcase-followup",
		Title:            "Closed eval-case follow-up",
		Summary:          "closed summary",
		SourceEvalCaseID: evalCaseID,
		CreatedBy:        "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedCase.ID, "operator-followup"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	openCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-report-badcase-followup",
		Title:            "Open eval-case follow-up",
		Summary:          "open summary",
		SourceEvalCaseID: evalCaseID,
		CreatedBy:        "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), openCase, "operator-followup"); err != nil {
		t.Fatalf("AssignCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-badcase-followup")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.BadCases) != 1 {
		t.Fatalf("len(got.BadCases) = %d, want 1", len(got.BadCases))
	}
	badCase := got.BadCases[0]
	if badCase.EvalCaseID != evalCaseID {
		t.Fatalf("BadCases[0].EvalCaseID = %q, want %q", badCase.EvalCaseID, evalCaseID)
	}
	if badCase.FollowUpCaseCount != 2 {
		t.Fatalf("BadCases[0].FollowUpCaseCount = %d, want 2", badCase.FollowUpCaseCount)
	}
	if badCase.OpenFollowUpCaseCount != 1 {
		t.Fatalf("BadCases[0].OpenFollowUpCaseCount = %d, want 1", badCase.OpenFollowUpCaseCount)
	}
	if got.BadCaseWithoutOpenFollowUpCount != 0 {
		t.Fatalf("BadCaseWithoutOpenFollowUpCount = %d, want 0", got.BadCaseWithoutOpenFollowUpCount)
	}
	if badCase.LatestFollowUpCaseID != openCase.ID {
		t.Fatalf("BadCases[0].LatestFollowUpCaseID = %q, want %q", badCase.LatestFollowUpCaseID, openCase.ID)
	}
	if badCase.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("BadCases[0].LatestFollowUpCaseStatus = %q, want %q", badCase.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if badCase.PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("BadCases[0].PreferredFollowUpAction.Mode = %q, want %q", badCase.PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if badCase.PreferredFollowUpAction.CaseID != openCase.ID {
		t.Fatalf("BadCases[0].PreferredFollowUpAction.CaseID = %q, want %q", badCase.PreferredFollowUpAction.CaseID, openCase.ID)
	}
	if badCase.PreferredFollowUpAction.SourceEvalCaseID != evalCaseID {
		t.Fatalf("BadCases[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", badCase.PreferredFollowUpAction.SourceEvalCaseID, evalCaseID)
	}
}

func TestGetEvalReportIncludesCompareFollowUpSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-detail", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Detail A", "Source Compare Detail A")
	rightReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-detail", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Detail B", "Source Compare Detail B")

	compareCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare-detail",
		Title:              "Left compare detail follow-up",
		Summary:            "left compare detail summary",
		SourceEvalReportID: leftReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "left",
		},
		CreatedBy: "operator-compare",
	})
	if err != nil {
		t.Fatalf("CreateCase(compareCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + leftReportID + "?tenant_id=tenant-eval-report-compare-detail")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.CompareFollowUpCaseCount != 1 || got.OpenCompareFollowUpCaseCount != 1 {
		t.Fatalf("compare follow-up counts = %#v, want count=1 open=1", got)
	}
	if got.LatestCompareFollowUpCaseID != compareCase.ID {
		t.Fatalf("LatestCompareFollowUpCaseID = %q, want %q", got.LatestCompareFollowUpCaseID, compareCase.ID)
	}
	if got.LatestCompareFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestCompareFollowUpCaseStatus = %q, want %q", got.LatestCompareFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if got.PreferredCompareFollowUpAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredCompareFollowUpAction.Mode = %q, want %q", got.PreferredCompareFollowUpAction.Mode, "open_existing_queue")
	}
	if got.PreferredCompareFollowUpAction.SourceEvalReportID != leftReportID {
		t.Fatalf("PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredCompareFollowUpAction.SourceEvalReportID, leftReportID)
	}
}

func TestGetEvalReportSupportsBadCaseNeedsFollowUpFilter(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-badcase-filter", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Bad Case Filter", "Source Bad Case Filter")

	report, err := reportService.GetEvalReport(context.Background(), reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	evalCaseID := report.BadCases[0].EvalCaseID
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-report-badcase-filter",
		Title:            "Open eval-case follow-up",
		Summary:          "open summary",
		SourceEvalCaseID: evalCaseID,
		CreatedBy:        "operator-followup",
	}); err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-badcase-filter&bad_case_needs_follow_up=true")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=true) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(true) = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var withFollowUp evalReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&withFollowUp); err != nil {
		t.Fatalf("Decode(withFollowUp) error = %v", err)
	}
	if len(withFollowUp.BadCases) != 1 {
		t.Fatalf("len(withFollowUp.BadCases) = %d, want 1", len(withFollowUp.BadCases))
	}
	if withFollowUp.BadCaseCount != 1 {
		t.Fatalf("withFollowUp.BadCaseCount = %d, want 1", withFollowUp.BadCaseCount)
	}
	if withFollowUp.BadCaseWithoutOpenFollowUpCount != 0 {
		t.Fatalf("withFollowUp.BadCaseWithoutOpenFollowUpCount = %d, want 0", withFollowUp.BadCaseWithoutOpenFollowUpCount)
	}
	if withFollowUp.BadCases[0].EvalCaseID != evalCaseID {
		t.Fatalf("withFollowUp.BadCases[0].EvalCaseID = %q, want %q", withFollowUp.BadCases[0].EvalCaseID, evalCaseID)
	}

	noFollowUpResp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-badcase-filter&bad_case_needs_follow_up=false")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=false) error = %v", err)
	}
	defer noFollowUpResp.Body.Close()
	if noFollowUpResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(false) = %d, want %d", noFollowUpResp.StatusCode, http.StatusOK)
	}
	var noFollowUp evalReportResponse
	if err := json.NewDecoder(noFollowUpResp.Body).Decode(&noFollowUp); err != nil {
		t.Fatalf("Decode(noFollowUp) error = %v", err)
	}
	if len(noFollowUp.BadCases) != 0 {
		t.Fatalf("len(noFollowUp.BadCases) = %d, want 0", len(noFollowUp.BadCases))
	}
	if noFollowUp.BadCaseCount != 1 {
		t.Fatalf("noFollowUp.BadCaseCount = %d, want 1", noFollowUp.BadCaseCount)
	}
	if noFollowUp.BadCaseWithoutOpenFollowUpCount != 0 {
		t.Fatalf("noFollowUp.BadCaseWithoutOpenFollowUpCount = %d, want 0", noFollowUp.BadCaseWithoutOpenFollowUpCount)
	}
}

func TestGetEvalReportRejectsInvalidBadCaseNeedsFollowUp(t *testing.T) {
	reportService, reportID := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusFailed, "failure detail")
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: casesvc.NewService(), EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-http&bad_case_needs_follow_up=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
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
	leftFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare",
		Title:              "Left follow-up",
		Summary:            "left summary",
		SourceEvalReportID: leftReportID,
		CreatedBy:          "operator-left",
	})
	if err != nil {
		t.Fatalf("CreateCase(leftFollowUp) error = %v", err)
	}
	rightFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare",
		Title:              "Right follow-up",
		Summary:            "right summary",
		SourceEvalReportID: rightReportID,
		CreatedBy:          "operator-right",
	})
	if err != nil {
		t.Fatalf("CreateCase(rightFollowUp) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
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
			ReportID                        string  `json:"report_id"`
			TenantID                        string  `json:"tenant_id"`
			RunID                           string  `json:"run_id"`
			DatasetID                       string  `json:"dataset_id"`
			DatasetName                     string  `json:"dataset_name"`
			RunStatus                       string  `json:"run_status"`
			Status                          string  `json:"status"`
			Summary                         string  `json:"summary"`
			TotalItems                      int     `json:"total_items"`
			RecordedResults                 int     `json:"recorded_results"`
			PassedItems                     int     `json:"passed_items"`
			FailedItems                     int     `json:"failed_items"`
			MissingResults                  int     `json:"missing_results"`
			AverageScore                    float64 `json:"average_score"`
			JudgeVersion                    string  `json:"judge_version"`
			VersionID                       string  `json:"version_id"`
			BadCaseCount                    int     `json:"bad_case_count"`
			BadCaseWithoutOpenFollowUpCount int     `json:"bad_case_without_open_follow_up_count"`
			FollowUpCaseCount               int     `json:"follow_up_case_count"`
			OpenFollowUpCaseCount           int     `json:"open_follow_up_case_count"`
			LatestFollowUpCaseID            string  `json:"latest_follow_up_case_id"`
			LatestFollowUpCaseStatus        string  `json:"latest_follow_up_case_status"`
			LinkedCaseSummary               struct {
				TotalCaseCount   int    `json:"total_case_count"`
				OpenCaseCount    int    `json:"open_case_count"`
				LatestCaseID     string `json:"latest_case_id"`
				LatestCaseStatus string `json:"latest_case_status"`
				LatestAssignedTo string `json:"latest_assigned_to"`
			} `json:"linked_case_summary"`
			CompareFollowUpCaseCount        int    `json:"compare_follow_up_case_count"`
			OpenCompareFollowUpCaseCount    int    `json:"open_compare_follow_up_case_count"`
			LatestCompareFollowUpCaseID     string `json:"latest_compare_follow_up_case_id"`
			LatestCompareFollowUpCaseStatus string `json:"latest_compare_follow_up_case_status"`
			PreferredCompareFollowUpAction  struct {
				Mode               string `json:"mode"`
				SourceEvalReportID string `json:"source_eval_report_id"`
			} `json:"preferred_compare_follow_up_action"`
		} `json:"left"`
		Right struct {
			ReportID                        string  `json:"report_id"`
			TenantID                        string  `json:"tenant_id"`
			RunID                           string  `json:"run_id"`
			DatasetID                       string  `json:"dataset_id"`
			DatasetName                     string  `json:"dataset_name"`
			RunStatus                       string  `json:"run_status"`
			Status                          string  `json:"status"`
			Summary                         string  `json:"summary"`
			TotalItems                      int     `json:"total_items"`
			RecordedResults                 int     `json:"recorded_results"`
			PassedItems                     int     `json:"passed_items"`
			FailedItems                     int     `json:"failed_items"`
			MissingResults                  int     `json:"missing_results"`
			AverageScore                    float64 `json:"average_score"`
			JudgeVersion                    string  `json:"judge_version"`
			VersionID                       string  `json:"version_id"`
			BadCaseCount                    int     `json:"bad_case_count"`
			BadCaseWithoutOpenFollowUpCount int     `json:"bad_case_without_open_follow_up_count"`
			FollowUpCaseCount               int     `json:"follow_up_case_count"`
			OpenFollowUpCaseCount           int     `json:"open_follow_up_case_count"`
			LatestFollowUpCaseID            string  `json:"latest_follow_up_case_id"`
			LatestFollowUpCaseStatus        string  `json:"latest_follow_up_case_status"`
			LinkedCaseSummary               struct {
				TotalCaseCount   int    `json:"total_case_count"`
				OpenCaseCount    int    `json:"open_case_count"`
				LatestCaseID     string `json:"latest_case_id"`
				LatestCaseStatus string `json:"latest_case_status"`
				LatestAssignedTo string `json:"latest_assigned_to"`
			} `json:"linked_case_summary"`
			CompareFollowUpCaseCount        int    `json:"compare_follow_up_case_count"`
			OpenCompareFollowUpCaseCount    int    `json:"open_compare_follow_up_case_count"`
			LatestCompareFollowUpCaseID     string `json:"latest_compare_follow_up_case_id"`
			LatestCompareFollowUpCaseStatus string `json:"latest_compare_follow_up_case_status"`
			PreferredCompareFollowUpAction  struct {
				Mode               string `json:"mode"`
				SourceEvalReportID string `json:"source_eval_report_id"`
			} `json:"preferred_compare_follow_up_action"`
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
	if got.Left.BadCaseWithoutOpenFollowUpCount != 0 {
		t.Fatalf("Left.BadCaseWithoutOpenFollowUpCount = %d, want 0", got.Left.BadCaseWithoutOpenFollowUpCount)
	}
	if got.Right.BadCaseWithoutOpenFollowUpCount != 1 {
		t.Fatalf("Right.BadCaseWithoutOpenFollowUpCount = %d, want 1", got.Right.BadCaseWithoutOpenFollowUpCount)
	}
	if got.Left.LatestFollowUpCaseID != leftFollowUp.ID {
		t.Fatalf("Left.LatestFollowUpCaseID = %q, want %q", got.Left.LatestFollowUpCaseID, leftFollowUp.ID)
	}
	if got.Left.FollowUpCaseCount != 1 || got.Left.OpenFollowUpCaseCount != 1 || got.Left.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("Left follow-up summary = %#v, want count=1 open=1 status=open", got.Left)
	}
	if got.Left.LinkedCaseSummary.TotalCaseCount != 1 || got.Left.LinkedCaseSummary.OpenCaseCount != 1 || got.Left.LinkedCaseSummary.LatestCaseID != leftFollowUp.ID {
		t.Fatalf("Left linked case summary = %#v, want 1 total / 1 open / latest %q", got.Left.LinkedCaseSummary, leftFollowUp.ID)
	}
	if got.Left.CompareFollowUpCaseCount != 0 || got.Left.OpenCompareFollowUpCaseCount != 0 || got.Left.LatestCompareFollowUpCaseID != "" || got.Left.LatestCompareFollowUpCaseStatus != "" {
		t.Fatalf("Left compare follow-up summary = %#v, want zero-value compare summary before compare-derived cases exist", got.Left)
	}
	if got.Left.PreferredCompareFollowUpAction.Mode != "create" {
		t.Fatalf("Left.PreferredCompareFollowUpAction.Mode = %q, want %q", got.Left.PreferredCompareFollowUpAction.Mode, "create")
	}
	if got.Left.PreferredCompareFollowUpAction.SourceEvalReportID != leftReportID {
		t.Fatalf("Left.PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.Left.PreferredCompareFollowUpAction.SourceEvalReportID, leftReportID)
	}
	if got.Right.LatestFollowUpCaseID != rightFollowUp.ID {
		t.Fatalf("Right.LatestFollowUpCaseID = %q, want %q", got.Right.LatestFollowUpCaseID, rightFollowUp.ID)
	}
	if got.Right.FollowUpCaseCount != 1 || got.Right.OpenFollowUpCaseCount != 1 || got.Right.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("Right follow-up summary = %#v, want count=1 open=1 status=open", got.Right)
	}
	if got.Right.LinkedCaseSummary.TotalCaseCount != 1 || got.Right.LinkedCaseSummary.OpenCaseCount != 1 || got.Right.LinkedCaseSummary.LatestCaseID != rightFollowUp.ID {
		t.Fatalf("Right linked case summary = %#v, want 1 total / 1 open / latest %q", got.Right.LinkedCaseSummary, rightFollowUp.ID)
	}
	if got.Right.CompareFollowUpCaseCount != 0 || got.Right.OpenCompareFollowUpCaseCount != 0 || got.Right.LatestCompareFollowUpCaseID != "" || got.Right.LatestCompareFollowUpCaseStatus != "" {
		t.Fatalf("Right compare follow-up summary = %#v, want zero-value compare summary before compare-derived cases exist", got.Right)
	}
	if got.Right.PreferredCompareFollowUpAction.Mode != "create" {
		t.Fatalf("Right.PreferredCompareFollowUpAction.Mode = %q, want %q", got.Right.PreferredCompareFollowUpAction.Mode, "create")
	}
	if got.Right.PreferredCompareFollowUpAction.SourceEvalReportID != rightReportID {
		t.Fatalf("Right.PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.Right.PreferredCompareFollowUpAction.SourceEvalReportID, rightReportID)
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

func TestCompareEvalReportsIncludesCompareFollowUpSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-origin", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare A", "Source Left")
	rightReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-origin", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare B", "Source Right")
	leftCompareFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare-origin",
		Title:              "Left compare-derived follow-up",
		Summary:            "left compare-derived summary",
		SourceEvalReportID: leftReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "left",
		},
		CreatedBy: "operator-left",
	})
	if err != nil {
		t.Fatalf("CreateCase(leftCompareFollowUp) error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	rightCompareFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare-origin",
		Title:              "Right compare-derived follow-up",
		Summary:            "right compare-derived summary",
		SourceEvalReportID: rightReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "right",
		},
		CreatedBy: "operator-right",
	})
	if err != nil {
		t.Fatalf("CreateCase(rightCompareFollowUp) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-report-compare?tenant_id=tenant-eval-report-compare-origin&left_report_id=" + leftReportID + "&right_report_id=" + rightReportID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got struct {
		Left struct {
			CompareFollowUpCaseCount        int    `json:"compare_follow_up_case_count"`
			OpenCompareFollowUpCaseCount    int    `json:"open_compare_follow_up_case_count"`
			LatestCompareFollowUpCaseID     string `json:"latest_compare_follow_up_case_id"`
			LatestCompareFollowUpCaseStatus string `json:"latest_compare_follow_up_case_status"`
			PreferredCompareFollowUpAction  struct {
				Mode               string `json:"mode"`
				SourceEvalReportID string `json:"source_eval_report_id"`
			} `json:"preferred_compare_follow_up_action"`
		} `json:"left"`
		Right struct {
			CompareFollowUpCaseCount        int    `json:"compare_follow_up_case_count"`
			OpenCompareFollowUpCaseCount    int    `json:"open_compare_follow_up_case_count"`
			LatestCompareFollowUpCaseID     string `json:"latest_compare_follow_up_case_id"`
			LatestCompareFollowUpCaseStatus string `json:"latest_compare_follow_up_case_status"`
			PreferredCompareFollowUpAction  struct {
				Mode               string `json:"mode"`
				SourceEvalReportID string `json:"source_eval_report_id"`
			} `json:"preferred_compare_follow_up_action"`
		} `json:"right"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Left.CompareFollowUpCaseCount != 1 || got.Left.OpenCompareFollowUpCaseCount != 1 || got.Left.LatestCompareFollowUpCaseID != leftCompareFollowUp.ID || got.Left.LatestCompareFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("Left compare follow-up summary = %#v, want count=1 open=1 latest=%q status=%q", got.Left, leftCompareFollowUp.ID, casesvc.StatusOpen)
	}
	if got.Left.PreferredCompareFollowUpAction.Mode != "open_existing_queue" {
		t.Fatalf("Left.PreferredCompareFollowUpAction.Mode = %q, want %q", got.Left.PreferredCompareFollowUpAction.Mode, "open_existing_queue")
	}
	if got.Left.PreferredCompareFollowUpAction.SourceEvalReportID != leftReportID {
		t.Fatalf("Left.PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.Left.PreferredCompareFollowUpAction.SourceEvalReportID, leftReportID)
	}
	if got.Right.CompareFollowUpCaseCount != 1 || got.Right.OpenCompareFollowUpCaseCount != 1 || got.Right.LatestCompareFollowUpCaseID != rightCompareFollowUp.ID || got.Right.LatestCompareFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("Right compare follow-up summary = %#v, want count=1 open=1 latest=%q status=%q", got.Right, rightCompareFollowUp.ID, casesvc.StatusOpen)
	}
	if got.Right.PreferredCompareFollowUpAction.Mode != "open_existing_queue" {
		t.Fatalf("Right.PreferredCompareFollowUpAction.Mode = %q, want %q", got.Right.PreferredCompareFollowUpAction.Mode, "open_existing_queue")
	}
	if got.Right.PreferredCompareFollowUpAction.SourceEvalReportID != rightReportID {
		t.Fatalf("Right.PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.Right.PreferredCompareFollowUpAction.SourceEvalReportID, rightReportID)
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

func TestListEvalReportsSupportsReportIDFilter(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	firstReportID := materializeEvalRunReport(t, "tenant-eval-report-filter", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset One", "Source One")
	_ = materializeEvalRunReport(t, "tenant-eval-report-filter", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Two", "Source Two")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{EvalReports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-filter&report_id=" + firstReportID + "&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode(page) error = %v", err)
	}
	if len(page.Reports) != 1 {
		t.Fatalf("len(page.Reports) = %d, want 1", len(page.Reports))
	}
	if page.Reports[0].ReportID != firstReportID {
		t.Fatalf("page.Reports[0].ReportID = %q, want %q", page.Reports[0].ReportID, firstReportID)
	}
}

func TestListEvalReportsIncludesFollowUpCaseSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-followup", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Followup", "Regression Source")

	closedCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-followup",
		Title:              "Closed follow-up",
		Summary:            "closed summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedCase.ID, "operator-followup"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	openCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-followup",
		Title:              "Open follow-up",
		Summary:            "open summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-followup",
	})
	if err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), openCase, "operator-followup"); err != nil {
		t.Fatalf("AssignCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-followup&status=ready&limit=10")
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
	if len(page.Reports) != 1 {
		t.Fatalf("len(page.Reports) = %d, want 1", len(page.Reports))
	}
	got := page.Reports[0]
	if got.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseCount = %d, want 2", got.FollowUpCaseCount)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 1", got.OpenFollowUpCaseCount)
	}
	if got.LatestFollowUpCaseID == "" {
		t.Fatal("LatestFollowUpCaseID is empty")
	}
	if got.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", got.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if got.BadCaseWithoutOpenFollowUpCount != 1 {
		t.Fatalf("BadCaseWithoutOpenFollowUpCount = %d, want 1", got.BadCaseWithoutOpenFollowUpCount)
	}
	if got.PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if got.PreferredFollowUpAction.CaseID != openCase.ID {
		t.Fatalf("PreferredFollowUpAction.CaseID = %q, want %q", got.PreferredFollowUpAction.CaseID, openCase.ID)
	}
	if got.PreferredFollowUpAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredFollowUpAction.SourceEvalReportID, reportID)
	}
}

func TestListEvalReportsIncludesCompareFollowUpSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-list", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare List A", "Source Compare List A")
	rightReportID := materializeEvalRunReport(t, "tenant-eval-report-compare-list", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare List B", "Source Compare List B")

	compareCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-compare-list",
		Title:              "Left compare list follow-up",
		Summary:            "left compare list summary",
		SourceEvalReportID: leftReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftReportID,
			RightEvalReportID: rightReportID,
			SelectedSide:      "left",
		},
		CreatedBy: "operator-compare",
	})
	if err != nil {
		t.Fatalf("CreateCase(compareCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-compare-list&status=ready&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Reports) != 2 {
		t.Fatalf("len(page.Reports) = %d, want 2", len(page.Reports))
	}
	var leftFound bool
	for _, got := range page.Reports {
		if got.ReportID != leftReportID {
			continue
		}
		leftFound = true
		if got.CompareFollowUpCaseCount != 1 || got.OpenCompareFollowUpCaseCount != 1 {
			t.Fatalf("compare follow-up counts = %#v, want count=1 open=1", got)
		}
		if got.LatestCompareFollowUpCaseID != compareCase.ID {
			t.Fatalf("LatestCompareFollowUpCaseID = %q, want %q", got.LatestCompareFollowUpCaseID, compareCase.ID)
		}
		if got.LatestCompareFollowUpCaseStatus != casesvc.StatusOpen {
			t.Fatalf("LatestCompareFollowUpCaseStatus = %q, want %q", got.LatestCompareFollowUpCaseStatus, casesvc.StatusOpen)
		}
		if got.PreferredCompareFollowUpAction.Mode != "open_existing_queue" {
			t.Fatalf("PreferredCompareFollowUpAction.Mode = %q, want %q", got.PreferredCompareFollowUpAction.Mode, "open_existing_queue")
		}
		if got.PreferredCompareFollowUpAction.SourceEvalReportID != leftReportID {
			t.Fatalf("PreferredCompareFollowUpAction.SourceEvalReportID = %q, want %q", got.PreferredCompareFollowUpAction.SourceEvalReportID, leftReportID)
		}
	}
	if !leftFound {
		t.Fatalf("left report %q not found in list response: %#v", leftReportID, page.Reports)
	}
}

func TestListEvalReportsIncludesLinkedCaseSummary(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-linked-list", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Linked List", "Source Linked List")

	closedCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-linked-list",
		Title:              "Closed linked follow-up",
		Summary:            "closed linked summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-linked",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedCase.ID, "operator-linked"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	openCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-linked-list",
		Title:              "Open linked follow-up",
		Summary:            "open linked summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-linked",
	})
	if err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), openCase, "owner-linked"); err != nil {
		t.Fatalf("AssignCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-linked-list&status=ready&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Reports) != 1 {
		t.Fatalf("len(page.Reports) = %d, want 1", len(page.Reports))
	}
	got := page.Reports[0]
	if got.LinkedCaseSummary == nil {
		t.Fatal("LinkedCaseSummary is nil")
	}
	if got.LinkedCaseSummary.TotalCaseCount != 2 || got.LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary counts = %#v, want total=2 open=1", got.LinkedCaseSummary)
	}
	if got.LinkedCaseSummary.LatestCaseID != openCase.ID {
		t.Fatalf("LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, openCase.ID)
	}
	if got.LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestCaseStatus = %q, want %q", got.LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.LinkedCaseSummary.LatestAssignedTo != "owner-linked" {
		t.Fatalf("LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "owner-linked")
	}
	if got.PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredLinkedCaseAction.CaseID != openCase.ID {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want %q", got.PreferredLinkedCaseAction.CaseID, openCase.ID)
	}
	if got.PreferredLinkedCaseAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredLinkedCaseAction.SourceEvalReportID = %q, want %q", got.PreferredLinkedCaseAction.SourceEvalReportID, reportID)
	}
}

func TestGetEvalReportLinkedCaseActionPrefersQueueWhenLatestCaseClosed(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportID := materializeEvalRunReport(t, "tenant-eval-report-linked-queue", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Linked Queue", "Source Linked Queue")

	closedCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-linked-queue",
		Title:              "Only closed linked follow-up",
		Summary:            "closed linked summary",
		SourceEvalReportID: reportID,
		CreatedBy:          "operator-linked",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedCase.ID, "operator-linked"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports/" + reportID + "?tenant_id=tenant-eval-report-linked-queue")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.LinkedCaseSummary == nil {
		t.Fatal("LinkedCaseSummary is nil")
	}
	if got.LinkedCaseSummary.TotalCaseCount != 1 || got.LinkedCaseSummary.OpenCaseCount != 0 {
		t.Fatalf("LinkedCaseSummary = %#v, want total=1 open=0", got.LinkedCaseSummary)
	}
	if got.PreferredLinkedCaseAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "open_existing_queue")
	}
	if got.PreferredLinkedCaseAction.CaseID != "" {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want empty", got.PreferredLinkedCaseAction.CaseID)
	}
	if got.PreferredLinkedCaseAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredLinkedCaseAction.SourceEvalReportID = %q, want %q", got.PreferredLinkedCaseAction.SourceEvalReportID, reportID)
	}
}

func TestListEvalReportsSupportsBadCaseNeedsFollowUpFilter(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	reportWithGapID := materializeEvalRunReport(t, "tenant-eval-report-badcase-list", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset With Gap", "Source With Gap")
	reportWithoutGapID := materializeEvalRunReport(t, "tenant-eval-report-badcase-list", evalsvc.RunStatusFailed, "second failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Without Gap", "Source Without Gap")

	reportWithoutGap, err := reportService.GetEvalReport(context.Background(), reportWithoutGapID)
	if err != nil {
		t.Fatalf("GetEvalReport(reportWithoutGapID) error = %v", err)
	}
	if len(reportWithoutGap.BadCases) == 0 {
		t.Fatal("reportWithoutGap.BadCases is empty")
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-report-badcase-list",
		Title:            "Open bad-case follow-up",
		Summary:          "open summary",
		SourceEvalCaseID: reportWithoutGap.BadCases[0].EvalCaseID,
		CreatedBy:        "operator-followup",
	}); err != nil {
		t.Fatalf("CreateCase(open bad-case follow-up) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-badcase-list&bad_case_needs_follow_up=true")
	if err != nil {
		t.Fatalf("Get(bad_case_needs_follow_up=true) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(true) = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var withGap listEvalReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&withGap); err != nil {
		t.Fatalf("Decode(withGap) error = %v", err)
	}
	if len(withGap.Reports) != 1 {
		t.Fatalf("len(withGap.Reports) = %d, want 1", len(withGap.Reports))
	}
	if withGap.Reports[0].ReportID != reportWithGapID {
		t.Fatalf("withGap.Reports[0].ReportID = %q, want %q", withGap.Reports[0].ReportID, reportWithGapID)
	}
	if withGap.Reports[0].BadCaseWithoutOpenFollowUpCount != 1 {
		t.Fatalf("withGap.Reports[0].BadCaseWithoutOpenFollowUpCount = %d, want 1", withGap.Reports[0].BadCaseWithoutOpenFollowUpCount)
	}

	noGapResp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-badcase-list&bad_case_needs_follow_up=false")
	if err != nil {
		t.Fatalf("Get(bad_case_needs_follow_up=false) error = %v", err)
	}
	defer noGapResp.Body.Close()
	if noGapResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(false) = %d, want %d", noGapResp.StatusCode, http.StatusOK)
	}
	var withoutGap listEvalReportsResponse
	if err := json.NewDecoder(noGapResp.Body).Decode(&withoutGap); err != nil {
		t.Fatalf("Decode(withoutGap) error = %v", err)
	}
	if len(withoutGap.Reports) != 1 {
		t.Fatalf("len(withoutGap.Reports) = %d, want 1", len(withoutGap.Reports))
	}
	if withoutGap.Reports[0].ReportID != reportWithoutGapID {
		t.Fatalf("withoutGap.Reports[0].ReportID = %q, want %q", withoutGap.Reports[0].ReportID, reportWithoutGapID)
	}
	if withoutGap.Reports[0].BadCaseWithoutOpenFollowUpCount != 0 {
		t.Fatalf("withoutGap.Reports[0].BadCaseWithoutOpenFollowUpCount = %d, want 0", withoutGap.Reports[0].BadCaseWithoutOpenFollowUpCount)
	}
}

func TestListEvalReportsRejectsInvalidBadCaseNeedsFollowUp(t *testing.T) {
	reportService, _ := buildEvalReportFixture(t, "tenant-eval-report-http", evalsvc.RunStatusFailed, "failure detail")
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       casesvc.NewService(),
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-http&bad_case_needs_follow_up=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestListEvalReportsSupportsNeedsFollowUpFilter(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	withFollowUpID := materializeEvalRunReport(t, "tenant-eval-report-needs-followup", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset With Follow-up", "Source With Follow-up")
	withoutFollowUpID := materializeEvalRunReport(t, "tenant-eval-report-needs-followup", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Without Follow-up", "Source Without Follow-up")

	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-needs-followup",
		Title:              "Open follow-up",
		Summary:            "open summary",
		SourceEvalReportID: withFollowUpID,
		CreatedBy:          "operator-followup",
	}); err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-needs-followup&status=ready&needs_follow_up=true&limit=10")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=true) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(true) = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var followUpPage listEvalReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&followUpPage); err != nil {
		t.Fatalf("Decode(followUpPage) error = %v", err)
	}
	if len(followUpPage.Reports) != 1 {
		t.Fatalf("len(followUpPage.Reports) = %d, want 1", len(followUpPage.Reports))
	}
	if followUpPage.Reports[0].ReportID != withFollowUpID {
		t.Fatalf("followUpPage.Reports[0].ReportID = %q, want %q", followUpPage.Reports[0].ReportID, withFollowUpID)
	}

	noFollowUpResp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-needs-followup&status=ready&needs_follow_up=false&limit=10")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=false) error = %v", err)
	}
	defer noFollowUpResp.Body.Close()
	if noFollowUpResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(false) = %d, want %d", noFollowUpResp.StatusCode, http.StatusOK)
	}

	var noFollowUpPage listEvalReportsResponse
	if err := json.NewDecoder(noFollowUpResp.Body).Decode(&noFollowUpPage); err != nil {
		t.Fatalf("Decode(noFollowUpPage) error = %v", err)
	}
	if len(noFollowUpPage.Reports) != 1 {
		t.Fatalf("len(noFollowUpPage.Reports) = %d, want 1", len(noFollowUpPage.Reports))
	}
	if noFollowUpPage.Reports[0].ReportID != withoutFollowUpID {
		t.Fatalf("noFollowUpPage.Reports[0].ReportID = %q, want %q", noFollowUpPage.Reports[0].ReportID, withoutFollowUpID)
	}
}

func TestListEvalReportsSupportsNeedsFollowUpPagination(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	firstReportID := materializeEvalRunReport(t, "tenant-eval-report-needs-followup-page", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset First", "Source First")
	secondReportID := materializeEvalRunReport(t, "tenant-eval-report-needs-followup-page", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Second", "Source Second")
	_ = materializeEvalRunReport(t, "tenant-eval-report-needs-followup-page", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset No Follow-up", "Source None")

	for _, reportID := range []string{firstReportID, secondReportID} {
		if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:           "tenant-eval-report-needs-followup-page",
			Title:              "Open follow-up",
			Summary:            "open summary",
			SourceEvalReportID: reportID,
			CreatedBy:          "operator-followup",
		}); err != nil {
			t.Fatalf("CreateCase(%s) error = %v", reportID, err)
		}
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	firstResp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-needs-followup-page&status=ready&needs_follow_up=true&limit=1")
	if err != nil {
		t.Fatalf("Get(first page) error = %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(first page) = %d, want %d", firstResp.StatusCode, http.StatusOK)
	}

	var firstPage listEvalReportsResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("Decode(firstPage) error = %v", err)
	}
	if len(firstPage.Reports) != 1 {
		t.Fatalf("len(firstPage.Reports) = %d, want 1", len(firstPage.Reports))
	}
	if !firstPage.HasMore {
		t.Fatal("firstPage.HasMore = false, want true")
	}
	if firstPage.NextOffset == nil || *firstPage.NextOffset != 1 {
		t.Fatalf("firstPage.NextOffset = %v, want 1", firstPage.NextOffset)
	}

	secondResp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval-report-needs-followup-page&status=ready&needs_follow_up=true&limit=1&offset=1")
	if err != nil {
		t.Fatalf("Get(second page) error = %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(second page) = %d, want %d", secondResp.StatusCode, http.StatusOK)
	}

	var secondPage listEvalReportsResponse
	if err := json.NewDecoder(secondResp.Body).Decode(&secondPage); err != nil {
		t.Fatalf("Decode(secondPage) error = %v", err)
	}
	if len(secondPage.Reports) != 1 {
		t.Fatalf("len(secondPage.Reports) = %d, want 1", len(secondPage.Reports))
	}
	if secondPage.HasMore {
		t.Fatal("secondPage.HasMore = true, want false")
	}
	if secondPage.Reports[0].ReportID == firstPage.Reports[0].ReportID {
		t.Fatalf("second page repeated report_id %q", secondPage.Reports[0].ReportID)
	}
}

func TestListEvalReportsRejectsInvalidNeedsFollowUp(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-reports?tenant_id=tenant-eval&needs_follow_up=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
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
