package httpapi

import (
	"bytes"
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

func TestCreateAndGetEvalDatasetEndpoint(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	createdCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:       "tenant-dataset",
		Title:          "Dataset source",
		Summary:        "Promote into dataset.",
		SourceTaskID:   "task-dataset-1",
		SourceReportID: "report-dataset-1",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: createdCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset","name":"Draft dataset","eval_case_ids":["`+evalCase.ID+`"],"created_by":"operator-1"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Contains(body, []byte(`"recent_runs":[]`)) {
		t.Fatalf("create response body = %s, want recent_runs as an empty array", string(body))
	}

	var created evalDatasetResponse
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if created.DatasetID == "" {
		t.Fatal("dataset_id is empty")
	}
	if len(created.Items) != 1 || created.Items[0].EvalCaseID != evalCase.ID {
		t.Fatalf("Items = %#v, want one eval case item", created.Items)
	}
	if created.Items[0].PreferredLinkedCaseAction.Mode != "none" {
		t.Fatalf("created.Items[0].PreferredLinkedCaseAction.Mode = %q, want %q", created.Items[0].PreferredLinkedCaseAction.Mode, "none")
	}
	if created.Items[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("created.Items[0].PreferredFollowUpAction.Mode = %q, want %q", created.Items[0].PreferredFollowUpAction.Mode, "create")
	}
	if created.Items[0].PreferredPrimaryAction.Mode != "create" {
		t.Fatalf("created.Items[0].PreferredPrimaryAction.Mode = %q, want %q", created.Items[0].PreferredPrimaryAction.Mode, "create")
	}

	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:         "tenant-dataset",
		Title:            "Eval dataset item follow-up",
		SourceEvalCaseID: evalCase.ID,
	})
	if err != nil {
		t.Fatalf("CreateCase(item follow-up) error = %v", err)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + created.DatasetID + "?tenant_id=tenant-dataset")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}
	var got evalDatasetResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode(get) error = %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("len(got.Items) = %d, want 1", len(got.Items))
	}
	if got.Items[0].LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("got.Items[0].LinkedCaseSummary.TotalCaseCount = %d, want 1", got.Items[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("got.Items[0].LinkedCaseSummary.OpenCaseCount = %d, want 1", got.Items[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("got.Items[0].LinkedCaseSummary.LatestCaseID = %q, want %q", got.Items[0].LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.Items[0].PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("got.Items[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.Items[0].PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.Items[0].PreferredLinkedCaseAction.CaseID != followUpCase.ID {
		t.Fatalf("got.Items[0].PreferredLinkedCaseAction.CaseID = %q, want %q", got.Items[0].PreferredLinkedCaseAction.CaseID, followUpCase.ID)
	}
	if got.Items[0].PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("got.Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if got.Items[0].PreferredFollowUpAction.CaseID != followUpCase.ID {
		t.Fatalf("got.Items[0].PreferredFollowUpAction.CaseID = %q, want %q", got.Items[0].PreferredFollowUpAction.CaseID, followUpCase.ID)
	}
	if got.Items[0].PreferredPrimaryAction.Mode != "open_existing_case" {
		t.Fatalf("got.Items[0].PreferredPrimaryAction.Mode = %q, want %q", got.Items[0].PreferredPrimaryAction.Mode, "open_existing_case")
	}
	if got.Items[0].PreferredPrimaryAction.CaseID != followUpCase.ID {
		t.Fatalf("got.Items[0].PreferredPrimaryAction.CaseID = %q, want %q", got.Items[0].PreferredPrimaryAction.CaseID, followUpCase.ID)
	}
}

func TestCreateEvalDatasetEndpointRejectsCrossTenantEvalCase(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	createdCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-a",
		Title:    "Cross tenant eval",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-a",
		SourceCaseID: createdCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-b","eval_case_ids":["`+evalCase.ID+`"]}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestGetEvalDatasetItemPreferredFollowUpActionCreatesWhenLatestCaseClosed(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-queue",
		Title:    "Dataset queue source",
	})
	if err != nil {
		t.Fatalf("CreateCase(source) error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-queue",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-queue",
		Name:        "Queue dataset",
		EvalCaseIDs: []string{evalCase.ID},
		CreatedBy:   "operator-queue",
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:         "tenant-dataset-queue",
		Title:            "Closed eval-case follow-up",
		SourceEvalCaseID: evalCase.ID,
	})
	if err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, followUpCase.ID, "closer-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + dataset.ID + "?tenant_id=tenant-dataset-queue")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("len(got.Items) = %d, want 1", len(got.Items))
	}
	if got.Items[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("got.Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.Items[0].PreferredFollowUpAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("got.Items[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", got.Items[0].PreferredFollowUpAction.SourceEvalCaseID, evalCase.ID)
	}
	if got.Items[0].PreferredPrimaryAction.Mode != "open_existing_queue" {
		t.Fatalf("got.Items[0].PreferredPrimaryAction.Mode = %q, want %q", got.Items[0].PreferredPrimaryAction.Mode, "open_existing_queue")
	}
	if got.Items[0].PreferredPrimaryAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("got.Items[0].PreferredPrimaryAction.SourceEvalCaseID = %q, want %q", got.Items[0].PreferredPrimaryAction.SourceEvalCaseID, evalCase.ID)
	}
}

func TestListEvalDatasetsEndpointSupportsFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	caseA, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source A",
	})
	if err != nil {
		t.Fatalf("CreateCase(caseA) error = %v", err)
	}
	evalA, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: caseA.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalA) error = %v", err)
	}

	caseB, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source B",
	})
	if err != nil {
		t.Fatalf("CreateCase(caseB) error = %v", err)
	}
	evalB, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: caseB.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalB) error = %v", err)
	}

	if _, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		Name:        "Dataset A",
		EvalCaseIDs: []string{evalA.ID},
		CreatedBy:   "operator-a",
	}); err != nil {
		t.Fatalf("CreateDataset(first) error = %v", err)
	}
	second, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		Name:        "Dataset B",
		EvalCaseIDs: []string{evalB.ID},
		CreatedBy:   "operator-b",
	})
	if err != nil {
		t.Fatalf("CreateDataset(second) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset&created_by=operator-b&limit=1")
	if err != nil {
		t.Fatalf("Get(filtered list) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalDatasetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode(filtered list) error = %v", err)
	}
	if len(page.Datasets) != 1 || page.Datasets[0].DatasetID != second.ID {
		t.Fatalf("Datasets = %#v, want only %q", page.Datasets, second.ID)
	}
	if page.Datasets[0].ItemCount != 1 {
		t.Fatalf("ItemCount = %d, want 1", page.Datasets[0].ItemCount)
	}

	resp, err = http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset&limit=1&offset=1")
	if err != nil {
		t.Fatalf("Get(paginated list) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode(paginated list) error = %v", err)
	}
	if len(page.Datasets) != 1 {
		t.Fatalf("len(Datasets) = %d, want 1", len(page.Datasets))
	}
}

func TestListEvalDatasetsEndpointRequiresTenant(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestListEvalDatasetsEndpointIncludesLatestRunSummary(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-dataset-summary", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Summary", "Dataset Source")
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if _, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-detail",
		Title:              "Dataset detail follow-up",
		SourceEvalReportID: reportID,
	}); err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
	}
	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-summary",
		Title:              "Dataset follow-up",
		SourceEvalReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
	}
	followUpCase, err = caseService.AssignCase(ctx, followUpCase, "dataset-operator")
	if err != nil {
		t.Fatalf("AssignCase(follow-up) error = %v", err)
	}
	runBackedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:        "tenant-dataset-summary",
		Title:           "Dataset run-backed follow-up",
		SourceEvalRunID: reportItem.RunID,
	})
	if err != nil {
		t.Fatalf("CreateCase(run-backed) error = %v", err)
	}
	runBackedCase, err = caseService.AssignCase(ctx, runBackedCase, "run-backed-operator")
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

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset-summary&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalDatasetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Datasets) != 1 {
		t.Fatalf("len(page.Datasets) = %d, want 1", len(page.Datasets))
	}
	got := page.Datasets[0]
	if got.DatasetID != reportItem.DatasetID {
		t.Fatalf("DatasetID = %q, want %q", got.DatasetID, reportItem.DatasetID)
	}
	if got.LatestRunID != reportItem.RunID {
		t.Fatalf("LatestRunID = %q, want %q", got.LatestRunID, reportItem.RunID)
	}
	if got.LatestRunStatus != evalsvc.RunStatusFailed {
		t.Fatalf("LatestRunStatus = %q, want %q", got.LatestRunStatus, evalsvc.RunStatusFailed)
	}
	if got.LatestReportID != reportID {
		t.Fatalf("LatestReportID = %q, want %q", got.LatestReportID, reportID)
	}
	if got.LatestReportStatus != evalsvc.EvalReportStatusReady {
		t.Fatalf("LatestReportStatus = %q, want %q", got.LatestReportStatus, evalsvc.EvalReportStatusReady)
	}
	if got.UnresolvedFollowUpCount != 1 {
		t.Fatalf("UnresolvedFollowUpCount = %d, want 1", got.UnresolvedFollowUpCount)
	}
	if !got.NeedsFollowUp {
		t.Fatal("NeedsFollowUp = false, want true")
	}
	if got.PreferredFollowUpAction.Mode != "open_latest_report_queue" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "open_latest_report_queue")
	}
	if got.PreferredFollowUpAction.ReportID != reportID {
		t.Fatalf("PreferredFollowUpAction.ReportID = %q, want %q", got.PreferredFollowUpAction.ReportID, reportID)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 1", got.OpenFollowUpCaseCount)
	}
	if got.PreferredCaseQueueAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredCaseQueueAction.Mode = %q, want %q", got.PreferredCaseQueueAction.Mode, "open_existing_case")
	}
	if got.PreferredCaseQueueAction.CaseID != followUpCase.ID {
		t.Fatalf("PreferredCaseQueueAction.CaseID = %q, want %q", got.PreferredCaseQueueAction.CaseID, followUpCase.ID)
	}
	if got.DatasetOpenFollowUpCaseCount != 1 {
		t.Fatalf("DatasetOpenFollowUpCaseCount = %d, want 1", got.DatasetOpenFollowUpCaseCount)
	}
	if got.PreferredDatasetCaseQueueAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredDatasetCaseQueueAction.Mode = %q, want %q", got.PreferredDatasetCaseQueueAction.Mode, "open_existing_case")
	}
	if got.PreferredDatasetCaseQueueAction.CaseID != followUpCase.ID {
		t.Fatalf("PreferredDatasetCaseQueueAction.CaseID = %q, want %q", got.PreferredDatasetCaseQueueAction.CaseID, followUpCase.ID)
	}
	if got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID != got.DatasetID {
		t.Fatalf("PreferredDatasetCaseQueueAction.SourceEvalDatasetID = %q, want %q", got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID, got.DatasetID)
	}
	if got.PreferredCaseHandoffAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredCaseHandoffAction.Mode = %q, want %q", got.PreferredCaseHandoffAction.Mode, "open_existing_case")
	}
	if got.PreferredCaseHandoffAction.CaseID != followUpCase.ID {
		t.Fatalf("PreferredCaseHandoffAction.CaseID = %q, want %q", got.PreferredCaseHandoffAction.CaseID, followUpCase.ID)
	}
	if got.PreferredCaseHandoffAction.SourceEvalDatasetID != got.DatasetID {
		t.Fatalf("PreferredCaseHandoffAction.SourceEvalDatasetID = %q, want %q", got.PreferredCaseHandoffAction.SourceEvalDatasetID, got.DatasetID)
	}
	if got.DatasetFollowUpCaseSummary.FollowUpCaseCount != 1 {
		t.Fatalf("DatasetFollowUpCaseSummary.FollowUpCaseCount = %d, want 1", got.DatasetFollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount != 1 {
		t.Fatalf("DatasetFollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 1", got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount != 0 {
		t.Fatalf("DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount = %d, want 0", got.DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.LatestFollowUpCaseID != followUpCase.ID {
		t.Fatalf("DatasetFollowUpCaseSummary.LatestFollowUpCaseID = %q, want %q", got.DatasetFollowUpCaseSummary.LatestFollowUpCaseID, followUpCase.ID)
	}
	if got.DatasetFollowUpCaseSummary.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("DatasetFollowUpCaseSummary.LatestFollowUpCaseStatus = %q, want %q", got.DatasetFollowUpCaseSummary.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if got.LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.TotalCaseCount = %d, want 1", got.LinkedCaseSummary.TotalCaseCount)
	}
	if got.LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.OpenCaseCount = %d, want 1", got.LinkedCaseSummary.OpenCaseCount)
	}
	if got.LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("LinkedCaseSummary.LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.LinkedCaseSummary.LatestAssignedTo != "dataset-operator" {
		t.Fatalf("LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "dataset-operator")
	}
	if got.RunBackedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("RunBackedCaseSummary.TotalCaseCount = %d, want 1", got.RunBackedCaseSummary.TotalCaseCount)
	}
	if got.RunBackedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("RunBackedCaseSummary.OpenCaseCount = %d, want 1", got.RunBackedCaseSummary.OpenCaseCount)
	}
	if got.RunBackedCaseSummary.LatestCaseID != runBackedCase.ID {
		t.Fatalf("RunBackedCaseSummary.LatestCaseID = %q, want %q", got.RunBackedCaseSummary.LatestCaseID, runBackedCase.ID)
	}
	if got.RunBackedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("RunBackedCaseSummary.LatestCaseStatus = %q, want %q", got.RunBackedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.RunBackedCaseSummary.LatestAssignedTo != "run-backed-operator" {
		t.Fatalf("RunBackedCaseSummary.LatestAssignedTo = %q, want %q", got.RunBackedCaseSummary.LatestAssignedTo, "run-backed-operator")
	}
	if got.PreferredRunBackedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredRunBackedCaseAction.Mode = %q, want %q", got.PreferredRunBackedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredRunBackedCaseAction.CaseID != runBackedCase.ID {
		t.Fatalf("PreferredRunBackedCaseAction.CaseID = %q, want %q", got.PreferredRunBackedCaseAction.CaseID, runBackedCase.ID)
	}
	if got.PreferredRunBackedCaseAction.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("PreferredRunBackedCaseAction.SourceEvalRunID = %q, want %q", got.PreferredRunBackedCaseAction.SourceEvalRunID, reportItem.RunID)
	}
}

func TestListEvalDatasetsEndpointSupportsNeedsFollowUpFilter(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-dataset-followup", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Needs Follow-up", "Dataset Needs Follow-up Source")
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-followup",
		Title:    "Dataset without run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-followup",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	withoutRun, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-followup",
		Name:        "Dataset Without Run",
		EvalCaseIDs: []string{evalCase.ID},
		CreatedBy:   "operator-dataset",
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	withFollowUpResp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset-followup&needs_follow_up=true&limit=10")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=true) error = %v", err)
	}
	defer withFollowUpResp.Body.Close()
	if withFollowUpResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(true) = %d, want %d", withFollowUpResp.StatusCode, http.StatusOK)
	}
	var withFollowUpPage listEvalDatasetsResponse
	if err := json.NewDecoder(withFollowUpResp.Body).Decode(&withFollowUpPage); err != nil {
		t.Fatalf("Decode(withFollowUpPage) error = %v", err)
	}
	if len(withFollowUpPage.Datasets) != 1 {
		t.Fatalf("len(withFollowUpPage.Datasets) = %d, want 1", len(withFollowUpPage.Datasets))
	}
	if withFollowUpPage.Datasets[0].DatasetID != reportItem.DatasetID {
		t.Fatalf("withFollowUpPage.Datasets[0].DatasetID = %q, want %q", withFollowUpPage.Datasets[0].DatasetID, reportItem.DatasetID)
	}

	withoutFollowUpResp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset-followup&needs_follow_up=false&limit=10")
	if err != nil {
		t.Fatalf("Get(needs_follow_up=false) error = %v", err)
	}
	defer withoutFollowUpResp.Body.Close()
	if withoutFollowUpResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(false) = %d, want %d", withoutFollowUpResp.StatusCode, http.StatusOK)
	}
	var withoutFollowUpPage listEvalDatasetsResponse
	if err := json.NewDecoder(withoutFollowUpResp.Body).Decode(&withoutFollowUpPage); err != nil {
		t.Fatalf("Decode(withoutFollowUpPage) error = %v", err)
	}
	if len(withoutFollowUpPage.Datasets) != 1 {
		t.Fatalf("len(withoutFollowUpPage.Datasets) = %d, want 1", len(withoutFollowUpPage.Datasets))
	}
	if withoutFollowUpPage.Datasets[0].DatasetID != withoutRun.ID {
		t.Fatalf("withoutFollowUpPage.Datasets[0].DatasetID = %q, want %q", withoutFollowUpPage.Datasets[0].DatasetID, withoutRun.ID)
	}
	if withoutFollowUpPage.Datasets[0].NeedsFollowUp {
		t.Fatal("withoutFollowUpPage.Datasets[0].NeedsFollowUp = true, want false")
	}
	if withoutFollowUpPage.Datasets[0].PreferredFollowUpAction.Mode != "none" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", withoutFollowUpPage.Datasets[0].PreferredFollowUpAction.Mode, "none")
	}
}

func TestListEvalDatasetsEndpointSupportsNeedsFollowUpPagination(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	firstReportID := materializeEvalRunReport(t, "tenant-dataset-pagination", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Older Match", "Dataset Older Match Source")
	secondReportID := materializeEvalRunReport(t, "tenant-dataset-pagination", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Newer Match", "Dataset Newer Match Source")

	firstReport, err := reportService.GetEvalReport(ctx, firstReportID)
	if err != nil {
		t.Fatalf("GetEvalReport(first) error = %v", err)
	}
	secondReport, err := reportService.GetEvalReport(ctx, secondReportID)
	if err != nil {
		t.Fatalf("GetEvalReport(second) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	firstPageResp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset-pagination&needs_follow_up=true&limit=1")
	if err != nil {
		t.Fatalf("Get(first page) error = %v", err)
	}
	defer firstPageResp.Body.Close()
	if firstPageResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(first page) = %d, want %d", firstPageResp.StatusCode, http.StatusOK)
	}
	var firstPage listEvalDatasetsResponse
	if err := json.NewDecoder(firstPageResp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("Decode(firstPage) error = %v", err)
	}
	if len(firstPage.Datasets) != 1 {
		t.Fatalf("len(firstPage.Datasets) = %d, want 1", len(firstPage.Datasets))
	}
	if firstPage.Datasets[0].DatasetID != secondReport.DatasetID {
		t.Fatalf("firstPage.Datasets[0].DatasetID = %q, want %q", firstPage.Datasets[0].DatasetID, secondReport.DatasetID)
	}
	if !firstPage.HasMore || firstPage.NextOffset == nil || *firstPage.NextOffset != 1 {
		t.Fatalf("firstPage pagination = %#v, want has_more with next_offset=1", firstPage)
	}

	secondPageResp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset-pagination&needs_follow_up=true&limit=1&offset=1")
	if err != nil {
		t.Fatalf("Get(second page) error = %v", err)
	}
	defer secondPageResp.Body.Close()
	if secondPageResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode(second page) = %d, want %d", secondPageResp.StatusCode, http.StatusOK)
	}
	var secondPage listEvalDatasetsResponse
	if err := json.NewDecoder(secondPageResp.Body).Decode(&secondPage); err != nil {
		t.Fatalf("Decode(secondPage) error = %v", err)
	}
	if len(secondPage.Datasets) != 1 {
		t.Fatalf("len(secondPage.Datasets) = %d, want 1", len(secondPage.Datasets))
	}
	if secondPage.Datasets[0].DatasetID != firstReport.DatasetID {
		t.Fatalf("secondPage.Datasets[0].DatasetID = %q, want %q", secondPage.Datasets[0].DatasetID, firstReport.DatasetID)
	}
	if secondPage.HasMore {
		t.Fatalf("secondPage.HasMore = true, want false")
	}
}

func TestListEvalDatasetsEndpointRejectsInvalidNeedsFollowUp(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets?tenant_id=tenant-dataset&needs_follow_up=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetEvalDatasetIncludesLatestRunSummary(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-dataset-detail", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Detail", "Dataset Detail Source")
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-detail",
		Title:              "Dataset detail follow-up",
		SourceEvalReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
	}
	followUpCase, err = caseService.AssignCase(ctx, followUpCase, "detail-operator")
	if err != nil {
		t.Fatalf("AssignCase(follow-up) error = %v", err)
	}
	runBackedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:        "tenant-dataset-detail",
		Title:           "Dataset detail run-backed follow-up",
		SourceEvalRunID: reportItem.RunID,
	})
	if err != nil {
		t.Fatalf("CreateCase(run-backed) error = %v", err)
	}
	runBackedCase, err = caseService.AssignCase(ctx, runBackedCase, "detail-run-operator")
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

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + reportItem.DatasetID + "?tenant_id=tenant-dataset-detail")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.LatestRunID != reportItem.RunID {
		t.Fatalf("LatestRunID = %q, want %q", got.LatestRunID, reportItem.RunID)
	}
	if got.LatestReportID != reportID {
		t.Fatalf("LatestReportID = %q, want %q", got.LatestReportID, reportID)
	}
	if got.UnresolvedFollowUpCount != 1 {
		t.Fatalf("UnresolvedFollowUpCount = %d, want 1", got.UnresolvedFollowUpCount)
	}
	if !got.NeedsFollowUp {
		t.Fatal("NeedsFollowUp = false, want true")
	}
	if got.PreferredFollowUpAction.Mode != "open_latest_report_queue" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "open_latest_report_queue")
	}
	if got.PreferredFollowUpAction.ReportID != reportID {
		t.Fatalf("PreferredFollowUpAction.ReportID = %q, want %q", got.PreferredFollowUpAction.ReportID, reportID)
	}
	if got.FollowUpCaseSummary.FollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseSummary.FollowUpCaseCount = %d, want 1", got.FollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.OpenFollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 1", got.FollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.ClosedFollowUpCaseCount != 0 {
		t.Fatalf("FollowUpCaseSummary.ClosedFollowUpCaseCount = %d, want 0", got.FollowUpCaseSummary.ClosedFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.LatestFollowUpCaseID == "" {
		t.Fatal("FollowUpCaseSummary.LatestFollowUpCaseID is empty, want follow-up case linkage")
	}
	if got.FollowUpCaseSummary.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("FollowUpCaseSummary.LatestFollowUpCaseStatus = %q, want %q", got.FollowUpCaseSummary.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 1", got.OpenFollowUpCaseCount)
	}
	if got.PreferredCaseQueueAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredCaseQueueAction.Mode = %q, want %q", got.PreferredCaseQueueAction.Mode, "open_existing_case")
	}
	if got.PreferredCaseQueueAction.CaseID == "" {
		t.Fatal("PreferredCaseQueueAction.CaseID is empty, want linked follow-up case")
	}
	if got.DatasetFollowUpCaseSummary.FollowUpCaseCount != 1 {
		t.Fatalf("DatasetFollowUpCaseSummary.FollowUpCaseCount = %d, want 1", got.DatasetFollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount != 1 {
		t.Fatalf("DatasetFollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 1", got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.DatasetOpenFollowUpCaseCount != 1 {
		t.Fatalf("DatasetOpenFollowUpCaseCount = %d, want 1", got.DatasetOpenFollowUpCaseCount)
	}
	if got.PreferredDatasetCaseQueueAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredDatasetCaseQueueAction.Mode = %q, want %q", got.PreferredDatasetCaseQueueAction.Mode, "open_existing_case")
	}
	if got.PreferredDatasetCaseQueueAction.CaseID == "" {
		t.Fatal("PreferredDatasetCaseQueueAction.CaseID is empty, want linked dataset-wide follow-up case")
	}
	if got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID != reportItem.DatasetID {
		t.Fatalf("PreferredDatasetCaseQueueAction.SourceEvalDatasetID = %q, want %q", got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID, reportItem.DatasetID)
	}
	if got.PreferredCaseHandoffAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredCaseHandoffAction.Mode = %q, want %q", got.PreferredCaseHandoffAction.Mode, "open_existing_case")
	}
	if got.PreferredCaseHandoffAction.CaseID == "" {
		t.Fatal("PreferredCaseHandoffAction.CaseID is empty, want linked dataset-wide follow-up case")
	}
	if got.PreferredCaseHandoffAction.SourceEvalDatasetID != reportItem.DatasetID {
		t.Fatalf("PreferredCaseHandoffAction.SourceEvalDatasetID = %q, want %q", got.PreferredCaseHandoffAction.SourceEvalDatasetID, reportItem.DatasetID)
	}
	if got.LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.TotalCaseCount = %d, want 1", got.LinkedCaseSummary.TotalCaseCount)
	}
	if got.LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.OpenCaseCount = %d, want 1", got.LinkedCaseSummary.OpenCaseCount)
	}
	if got.LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("LinkedCaseSummary.LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.LinkedCaseSummary.LatestAssignedTo != "detail-operator" {
		t.Fatalf("LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "detail-operator")
	}
	if got.RunBackedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("RunBackedCaseSummary.TotalCaseCount = %d, want 1", got.RunBackedCaseSummary.TotalCaseCount)
	}
	if got.RunBackedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("RunBackedCaseSummary.OpenCaseCount = %d, want 1", got.RunBackedCaseSummary.OpenCaseCount)
	}
	if got.RunBackedCaseSummary.LatestCaseID != runBackedCase.ID {
		t.Fatalf("RunBackedCaseSummary.LatestCaseID = %q, want %q", got.RunBackedCaseSummary.LatestCaseID, runBackedCase.ID)
	}
	if got.RunBackedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("RunBackedCaseSummary.LatestCaseStatus = %q, want %q", got.RunBackedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.RunBackedCaseSummary.LatestAssignedTo != "detail-run-operator" {
		t.Fatalf("RunBackedCaseSummary.LatestAssignedTo = %q, want %q", got.RunBackedCaseSummary.LatestAssignedTo, "detail-run-operator")
	}
	if got.PreferredRunBackedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredRunBackedCaseAction.Mode = %q, want %q", got.PreferredRunBackedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredRunBackedCaseAction.CaseID != runBackedCase.ID {
		t.Fatalf("PreferredRunBackedCaseAction.CaseID = %q, want %q", got.PreferredRunBackedCaseAction.CaseID, runBackedCase.ID)
	}
	if got.PreferredRunBackedCaseAction.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("PreferredRunBackedCaseAction.SourceEvalRunID = %q, want %q", got.PreferredRunBackedCaseAction.SourceEvalRunID, reportItem.RunID)
	}
	if len(got.RecentRuns) == 0 {
		t.Fatal("RecentRuns is empty, want latest run summary")
	}
	if got.RecentRuns[0].RunID != reportItem.RunID {
		t.Fatalf("RecentRuns[0].RunID = %q, want %q", got.RecentRuns[0].RunID, reportItem.RunID)
	}
	if got.RecentRuns[0].ReportID != reportID {
		t.Fatalf("RecentRuns[0].ReportID = %q, want %q", got.RecentRuns[0].ReportID, reportID)
	}
	if got.RecentRuns[0].PreferredFollowUpAction.Mode != "open_latest_report_queue" {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.Mode = %q, want %q", got.RecentRuns[0].PreferredFollowUpAction.Mode, "open_latest_report_queue")
	}
	if got.RecentRuns[0].PreferredFollowUpAction.ReportID != reportID {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.ReportID = %q, want %q", got.RecentRuns[0].PreferredFollowUpAction.ReportID, reportID)
	}
	if got.RecentRuns[0].PreferredFollowUpAction.RunID != "" {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.RunID = %q, want empty", got.RecentRuns[0].PreferredFollowUpAction.RunID)
	}
	if got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.TotalCaseCount = %d, want 1", got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.RecentRuns[0].LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.OpenCaseCount = %d, want 1", got.RecentRuns[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.RecentRuns[0].LinkedCaseSummary.LatestCaseID != runBackedCase.ID {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.LatestCaseID = %q, want %q", got.RecentRuns[0].LinkedCaseSummary.LatestCaseID, runBackedCase.ID)
	}
	if got.RecentRuns[0].LinkedCaseSummary.LatestAssignedTo != "detail-run-operator" {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.RecentRuns[0].LinkedCaseSummary.LatestAssignedTo, "detail-run-operator")
	}
	if got.RecentRuns[0].PreferredCaseAction.Mode != "open_existing_case" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.Mode = %q, want %q", got.RecentRuns[0].PreferredCaseAction.Mode, "open_existing_case")
	}
	if got.RecentRuns[0].PreferredCaseAction.CaseID != runBackedCase.ID {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.CaseID = %q, want %q", got.RecentRuns[0].PreferredCaseAction.CaseID, runBackedCase.ID)
	}
	if got.RecentRuns[0].PreferredCaseAction.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.SourceEvalRunID = %q, want %q", got.RecentRuns[0].PreferredCaseAction.SourceEvalRunID, reportItem.RunID)
	}
	if got.RecentRuns[0].PreferredPrimaryAction.Mode != "open_existing_case" {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.Mode = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.Mode, "open_existing_case")
	}
	if got.RecentRuns[0].PreferredPrimaryAction.CaseID != runBackedCase.ID {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.CaseID = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.CaseID, runBackedCase.ID)
	}
	if got.RecentRuns[0].PreferredPrimaryAction.RunID != reportItem.RunID {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.RunID = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.RunID, reportItem.RunID)
	}
}

func TestEvalDatasetFollowUpActionFallsBackToLatestRunQueue(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-run-fallback",
		Title:    "Dataset run fallback source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-run-fallback",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-run-fallback",
		Name:        "Dataset run fallback",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-dataset-run-fallback"}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}
	run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-dataset-run-fallback",
		DatasetID: dataset.ID,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if _, err := runService.MarkRunFailed(ctx, run.ID, "fault injection: dataset run failed"); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + dataset.ID + "?tenant_id=tenant-dataset-run-fallback")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.PreferredFollowUpAction.Mode != "open_latest_run_queue" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", got.PreferredFollowUpAction.Mode, "open_latest_run_queue")
	}
	if got.PreferredFollowUpAction.RunID != run.ID {
		t.Fatalf("PreferredFollowUpAction.RunID = %q, want %q", got.PreferredFollowUpAction.RunID, run.ID)
	}
	if got.PreferredFollowUpAction.ReportID != "" {
		t.Fatalf("PreferredFollowUpAction.ReportID = %q, want empty", got.PreferredFollowUpAction.ReportID)
	}
	if got.FollowUpCaseSummary.FollowUpCaseCount != 0 {
		t.Fatalf("FollowUpCaseSummary.FollowUpCaseCount = %d, want 0", got.FollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.OpenFollowUpCaseCount != 0 {
		t.Fatalf("FollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 0", got.FollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.ClosedFollowUpCaseCount != 0 {
		t.Fatalf("FollowUpCaseSummary.ClosedFollowUpCaseCount = %d, want 0", got.FollowUpCaseSummary.ClosedFollowUpCaseCount)
	}
	if got.OpenFollowUpCaseCount != 0 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 0", got.OpenFollowUpCaseCount)
	}
	if got.PreferredCaseQueueAction.Mode != "none" {
		t.Fatalf("PreferredCaseQueueAction.Mode = %q, want %q", got.PreferredCaseQueueAction.Mode, "none")
	}
	if got.PreferredRunBackedCaseAction.Mode != "none" {
		t.Fatalf("PreferredRunBackedCaseAction.Mode = %q, want %q", got.PreferredRunBackedCaseAction.Mode, "none")
	}
	if len(got.RecentRuns) == 0 {
		t.Fatal("RecentRuns is empty, want latest run summary")
	}
	if got.RecentRuns[0].RunID != run.ID {
		t.Fatalf("RecentRuns[0].RunID = %q, want %q", got.RecentRuns[0].RunID, run.ID)
	}
	if got.RecentRuns[0].ReportID != "" {
		t.Fatalf("RecentRuns[0].ReportID = %q, want empty", got.RecentRuns[0].ReportID)
	}
	if got.RecentRuns[0].PreferredFollowUpAction.Mode != "open_latest_run_queue" {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.Mode = %q, want %q", got.RecentRuns[0].PreferredFollowUpAction.Mode, "open_latest_run_queue")
	}
	if got.RecentRuns[0].PreferredFollowUpAction.RunID != run.ID {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.RunID = %q, want %q", got.RecentRuns[0].PreferredFollowUpAction.RunID, run.ID)
	}
	if got.RecentRuns[0].PreferredFollowUpAction.ReportID != "" {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.ReportID = %q, want empty", got.RecentRuns[0].PreferredFollowUpAction.ReportID)
	}
	if got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount != 0 {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.TotalCaseCount = %d, want 0", got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.RecentRuns[0].PreferredCaseAction.Mode != "none" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.Mode = %q, want %q", got.RecentRuns[0].PreferredCaseAction.Mode, "none")
	}
	if got.RecentRuns[0].PreferredPrimaryAction.Mode != "open_latest_run_queue" {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.Mode = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.Mode, "open_latest_run_queue")
	}
	if got.RecentRuns[0].PreferredPrimaryAction.RunID != run.ID {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.RunID = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.RunID, run.ID)
	}
	if got.RecentRuns[0].PreferredPrimaryAction.ReportID != "" {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.ReportID = %q, want empty", got.RecentRuns[0].PreferredPrimaryAction.ReportID)
	}
}

func TestEvalDatasetRecentRunPrimaryActionPrefersReportWhenNoQueueOrCase(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-dataset-report-primary", evalsvc.RunStatusSucceeded, "success detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Report Primary", "Dataset Report Primary Source")
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + reportItem.DatasetID + "?tenant_id=tenant-dataset-report-primary")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.RecentRuns) == 0 {
		t.Fatal("RecentRuns is empty, want latest run summary")
	}
	if got.RecentRuns[0].RunID != reportItem.RunID {
		t.Fatalf("RecentRuns[0].RunID = %q, want %q", got.RecentRuns[0].RunID, reportItem.RunID)
	}
	if got.RecentRuns[0].PreferredCaseAction.Mode != "none" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.Mode = %q, want %q", got.RecentRuns[0].PreferredCaseAction.Mode, "none")
	}
	if got.RecentRuns[0].PreferredFollowUpAction.Mode != "none" {
		t.Fatalf("RecentRuns[0].PreferredFollowUpAction.Mode = %q, want %q", got.RecentRuns[0].PreferredFollowUpAction.Mode, "none")
	}
	if got.RecentRuns[0].PreferredPrimaryAction.Mode != "open_report" {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.Mode = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.Mode, "open_report")
	}
	if got.RecentRuns[0].PreferredPrimaryAction.ReportID != reportID {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.ReportID = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.ReportID, reportID)
	}
	if got.RecentRuns[0].PreferredPrimaryAction.RunID != reportItem.RunID {
		t.Fatalf("RecentRuns[0].PreferredPrimaryAction.RunID = %q, want %q", got.RecentRuns[0].PreferredPrimaryAction.RunID, reportItem.RunID)
	}
}

func TestEvalDatasetCaseQueueActionPrefersQueueWhenLatestCaseIsClosed(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-dataset-case-queue", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Case Queue", "Dataset Case Queue Source")
	openCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-case-queue",
		Title:              "Older open follow-up",
		SourceEvalReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(open) error = %v", err)
	}
	closedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-case-queue",
		Title:              "Newer closed follow-up",
		SourceEvalReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, closedCase.ID, "operator-close"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + reportItem.DatasetID + "?tenant_id=tenant-dataset-case-queue")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want 1", got.OpenFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseSummary.FollowUpCaseCount = %d, want 2", got.FollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.OpenFollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 1", got.FollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.ClosedFollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseSummary.ClosedFollowUpCaseCount = %d, want 1", got.FollowUpCaseSummary.ClosedFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.LatestFollowUpCaseID != closedCase.ID {
		t.Fatalf("FollowUpCaseSummary.LatestFollowUpCaseID = %q, want %q", got.FollowUpCaseSummary.LatestFollowUpCaseID, closedCase.ID)
	}
	if got.FollowUpCaseSummary.LatestFollowUpCaseStatus != casesvc.StatusClosed {
		t.Fatalf("FollowUpCaseSummary.LatestFollowUpCaseStatus = %q, want %q", got.FollowUpCaseSummary.LatestFollowUpCaseStatus, casesvc.StatusClosed)
	}
	if got.PreferredCaseQueueAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredCaseQueueAction.Mode = %q, want %q", got.PreferredCaseQueueAction.Mode, "open_existing_queue")
	}
	if got.PreferredCaseQueueAction.CaseID != "" {
		t.Fatalf("PreferredCaseQueueAction.CaseID = %q, want empty", got.PreferredCaseQueueAction.CaseID)
	}
	if got.PreferredCaseQueueAction.SourceEvalReportID != reportID {
		t.Fatalf("PreferredCaseQueueAction.SourceEvalReportID = %q, want %q", got.PreferredCaseQueueAction.SourceEvalReportID, reportID)
	}
	if got.DatasetOpenFollowUpCaseCount != 1 {
		t.Fatalf("DatasetOpenFollowUpCaseCount = %d, want 1", got.DatasetOpenFollowUpCaseCount)
	}
	if got.PreferredDatasetCaseQueueAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredDatasetCaseQueueAction.Mode = %q, want %q", got.PreferredDatasetCaseQueueAction.Mode, "open_existing_queue")
	}
	if got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID != reportItem.DatasetID {
		t.Fatalf("PreferredDatasetCaseQueueAction.SourceEvalDatasetID = %q, want %q", got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID, reportItem.DatasetID)
	}
	if got.PreferredCaseHandoffAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredCaseHandoffAction.Mode = %q, want %q", got.PreferredCaseHandoffAction.Mode, "open_existing_queue")
	}
	if got.PreferredCaseHandoffAction.SourceEvalDatasetID != reportItem.DatasetID {
		t.Fatalf("PreferredCaseHandoffAction.SourceEvalDatasetID = %q, want %q", got.PreferredCaseHandoffAction.SourceEvalDatasetID, reportItem.DatasetID)
	}
	if openCase.ID == "" {
		t.Fatal("openCase.ID is empty")
	}
}

func TestEvalDatasetRunBackedCaseActionClearsWhenOnlyClosedCaseRemains(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(
		t,
		"tenant-dataset-run-backed-queue",
		evalsvc.RunStatusFailed,
		"failure detail",
		caseService,
		evalCaseService,
		datasetService,
		runService,
		reportService,
		"Dataset Run-backed Queue",
		"Dataset Run-backed Queue Source",
	)
	reportItem, err := reportService.GetEvalReport(ctx, reportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}

	closedRunCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:        "tenant-dataset-run-backed-queue",
		Title:           "Closed run-backed case",
		SourceEvalRunID: reportItem.RunID,
		CreatedBy:       "operator-closed",
	})
	if err != nil {
		t.Fatalf("CreateCase(closedRunCase) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, closedRunCase.ID, "operator-closed"); err != nil {
		t.Fatalf("CloseCase(closedRunCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + reportItem.DatasetID + "?tenant_id=tenant-dataset-run-backed-queue")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.RunBackedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("RunBackedCaseSummary.TotalCaseCount = %d, want 1", got.RunBackedCaseSummary.TotalCaseCount)
	}
	if got.RunBackedCaseSummary.OpenCaseCount != 0 {
		t.Fatalf("RunBackedCaseSummary.OpenCaseCount = %d, want 0", got.RunBackedCaseSummary.OpenCaseCount)
	}
	if got.RunBackedCaseSummary.LatestCaseID != closedRunCase.ID {
		t.Fatalf("RunBackedCaseSummary.LatestCaseID = %q, want %q", got.RunBackedCaseSummary.LatestCaseID, closedRunCase.ID)
	}
	if got.RunBackedCaseSummary.LatestCaseStatus != casesvc.StatusClosed {
		t.Fatalf("RunBackedCaseSummary.LatestCaseStatus = %q, want %q", got.RunBackedCaseSummary.LatestCaseStatus, casesvc.StatusClosed)
	}
	if got.PreferredRunBackedCaseAction.Mode != "none" {
		t.Fatalf("PreferredRunBackedCaseAction.Mode = %q, want %q", got.PreferredRunBackedCaseAction.Mode, "none")
	}
	if got.PreferredRunBackedCaseAction.CaseID != "" {
		t.Fatalf("PreferredRunBackedCaseAction.CaseID = %q, want empty", got.PreferredRunBackedCaseAction.CaseID)
	}
	if got.PreferredRunBackedCaseAction.SourceEvalRunID != "" {
		t.Fatalf("PreferredRunBackedCaseAction.SourceEvalRunID = %q, want empty", got.PreferredRunBackedCaseAction.SourceEvalRunID)
	}
	if len(got.RecentRuns) == 0 {
		t.Fatal("RecentRuns is empty, want latest run summary")
	}
	if got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.TotalCaseCount = %d, want 1", got.RecentRuns[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.RecentRuns[0].LinkedCaseSummary.OpenCaseCount != 0 {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.OpenCaseCount = %d, want 0", got.RecentRuns[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.RecentRuns[0].LinkedCaseSummary.LatestCaseID != closedRunCase.ID {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.LatestCaseID = %q, want %q", got.RecentRuns[0].LinkedCaseSummary.LatestCaseID, closedRunCase.ID)
	}
	if got.RecentRuns[0].LinkedCaseSummary.LatestCaseStatus != casesvc.StatusClosed {
		t.Fatalf("RecentRuns[0].LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.RecentRuns[0].LinkedCaseSummary.LatestCaseStatus, casesvc.StatusClosed)
	}
	if got.RecentRuns[0].PreferredCaseAction.Mode != "none" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.Mode = %q, want %q", got.RecentRuns[0].PreferredCaseAction.Mode, "none")
	}
	if got.RecentRuns[0].PreferredCaseAction.CaseID != "" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.CaseID = %q, want empty", got.RecentRuns[0].PreferredCaseAction.CaseID)
	}
	if got.RecentRuns[0].PreferredCaseAction.SourceEvalRunID != "" {
		t.Fatalf("RecentRuns[0].PreferredCaseAction.SourceEvalRunID = %q, want empty", got.RecentRuns[0].PreferredCaseAction.SourceEvalRunID)
	}
}

func TestGetEvalDatasetAggregatesDatasetWideFollowUpCaseSummaryAcrossReports(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-wide-summary",
		Title:    "Dataset Wide Summary Source",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-wide-summary",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-wide-summary",
		Name:        "Dataset Wide Summary",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-dataset-wide-summary"}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	firstRun, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-dataset-wide-summary",
		DatasetID: dataset.ID,
		CreatedBy: "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateRun(first) error = %v", err)
	}
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(first) error = %v", err)
	}
	if _, err := runService.MarkRunFailed(ctx, firstRun.ID, "older failure detail"); err != nil {
		t.Fatalf("MarkRunFailed(first) error = %v", err)
	}
	firstReportItem, err := reportService.MaterializeRunReport(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport(first) error = %v", err)
	}
	firstReportID := firstReportItem.ID

	secondRun, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-dataset-wide-summary",
		DatasetID: dataset.ID,
		CreatedBy: "operator-eval",
	})
	if err != nil {
		t.Fatalf("CreateRun(second) error = %v", err)
	}
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(second) error = %v", err)
	}
	if _, err := runService.MarkRunFailed(ctx, secondRun.ID, "newer failure detail"); err != nil {
		t.Fatalf("MarkRunFailed(second) error = %v", err)
	}
	secondReportItem, err := reportService.MaterializeRunReport(ctx, secondRun.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport(second) error = %v", err)
	}
	secondReportID := secondReportItem.ID

	firstReport, err := reportService.GetEvalReport(ctx, firstReportID)
	if err != nil {
		t.Fatalf("GetEvalReport(first) error = %v", err)
	}
	secondReport, err := reportService.GetEvalReport(ctx, secondReportID)
	if err != nil {
		t.Fatalf("GetEvalReport(second) error = %v", err)
	}
	if firstReport.DatasetID != secondReport.DatasetID {
		t.Fatalf("dataset mismatch: %q != %q", firstReport.DatasetID, secondReport.DatasetID)
	}

	if _, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-wide-summary",
		Title:              "Older open follow-up",
		SourceEvalReportID: firstReportID,
	}); err != nil {
		t.Fatalf("CreateCase(first open) error = %v", err)
	}
	latestOpenCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-wide-summary",
		Title:              "Latest open follow-up",
		SourceEvalReportID: secondReportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(second open) error = %v", err)
	}
	latestClosedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-wide-summary",
		Title:              "Latest closed follow-up",
		SourceEvalReportID: secondReportID,
	})
	if err != nil {
		t.Fatalf("CreateCase(second closed) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, latestClosedCase.ID, "operator-close"); err != nil {
		t.Fatalf("CloseCase(latestClosedCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + secondReport.DatasetID + "?tenant_id=tenant-dataset-wide-summary")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.FollowUpCaseSummary.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseSummary.FollowUpCaseCount = %d, want 2", got.FollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.OpenFollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 1", got.FollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.FollowUpCaseSummary.LatestFollowUpCaseID != latestClosedCase.ID {
		t.Fatalf("FollowUpCaseSummary.LatestFollowUpCaseID = %q, want %q", got.FollowUpCaseSummary.LatestFollowUpCaseID, latestClosedCase.ID)
	}
	if got.DatasetFollowUpCaseSummary.FollowUpCaseCount != 3 {
		t.Fatalf("DatasetFollowUpCaseSummary.FollowUpCaseCount = %d, want 3", got.DatasetFollowUpCaseSummary.FollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount != 2 {
		t.Fatalf("DatasetFollowUpCaseSummary.OpenFollowUpCaseCount = %d, want 2", got.DatasetFollowUpCaseSummary.OpenFollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount != 1 {
		t.Fatalf("DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount = %d, want 1", got.DatasetFollowUpCaseSummary.ClosedFollowUpCaseCount)
	}
	if got.DatasetFollowUpCaseSummary.LatestFollowUpCaseID != latestClosedCase.ID {
		t.Fatalf("DatasetFollowUpCaseSummary.LatestFollowUpCaseID = %q, want %q", got.DatasetFollowUpCaseSummary.LatestFollowUpCaseID, latestClosedCase.ID)
	}
	if got.DatasetOpenFollowUpCaseCount != 2 {
		t.Fatalf("DatasetOpenFollowUpCaseCount = %d, want 2", got.DatasetOpenFollowUpCaseCount)
	}
	if got.PreferredDatasetCaseQueueAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredDatasetCaseQueueAction.Mode = %q, want %q", got.PreferredDatasetCaseQueueAction.Mode, "open_existing_queue")
	}
	if got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID != secondReport.DatasetID {
		t.Fatalf("PreferredDatasetCaseQueueAction.SourceEvalDatasetID = %q, want %q", got.PreferredDatasetCaseQueueAction.SourceEvalDatasetID, secondReport.DatasetID)
	}
	if got.PreferredCaseHandoffAction.Mode != "open_existing_queue" {
		t.Fatalf("PreferredCaseHandoffAction.Mode = %q, want %q", got.PreferredCaseHandoffAction.Mode, "open_existing_queue")
	}
	if got.PreferredCaseHandoffAction.SourceEvalDatasetID != secondReport.DatasetID {
		t.Fatalf("PreferredCaseHandoffAction.SourceEvalDatasetID = %q, want %q", got.PreferredCaseHandoffAction.SourceEvalDatasetID, secondReport.DatasetID)
	}
	if latestOpenCase.ID == "" {
		t.Fatal("latestOpenCase.ID is empty")
	}
}

func TestCreateEvalDatasetEndpointRejectsDuplicateEvalCaseIDs(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	createdCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Duplicate dataset source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: createdCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset","eval_case_ids":["`+evalCase.ID+`","`+evalCase.ID+`"]}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestAddEvalDatasetItemEndpointAppendsMembership(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	firstCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source A",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	firstEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: firstCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}

	secondCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source B",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	secondEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: secondCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}

	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{firstEval.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/items", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset","eval_case_id":"`+secondEval.ID+`","added_by":"operator-1"}`))
	if err != nil {
		t.Fatalf("Post(add item) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(updated.Items) != 2 || updated.Items[1].EvalCaseID != secondEval.ID {
		t.Fatalf("Items = %#v, want appended second eval case", updated.Items)
	}
}

func TestAddEvalDatasetItemEndpointIsIdempotent(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	firstCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source A",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	firstEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: firstCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}

	secondCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source B",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	secondEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: secondCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}

	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{firstEval.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	body := `{"tenant_id":"tenant-dataset","eval_case_id":"` + secondEval.ID + `","added_by":"operator-1"}`
	for i := 0; i < 2; i++ {
		resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/items", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("Post(add item %d) error = %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var updated evalDatasetResponse
		if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
			resp.Body.Close()
			t.Fatalf("Decode(add item %d) error = %v", i+1, err)
		}
		resp.Body.Close()
		if len(updated.Items) != 2 {
			t.Fatalf("len(Items) after add %d = %d, want 2", i+1, len(updated.Items))
		}
	}
}

func TestAddEvalDatasetItemEndpointHidesCrossTenantDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	tenantADatasetCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-a",
		Title:    "Dataset source A",
	})
	if err != nil {
		t.Fatalf("CreateCase(tenant-a) error = %v", err)
	}
	tenantADatasetEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-a",
		SourceCaseID: tenantADatasetCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(tenant-a) error = %v", err)
	}
	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-a",
		EvalCaseIDs: []string{tenantADatasetEval.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	tenantBEvalCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-b",
		Title:    "Dataset source B",
	})
	if err != nil {
		t.Fatalf("CreateCase(tenant-b) error = %v", err)
	}
	tenantBEval, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-b",
		SourceCaseID: tenantBEvalCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(tenant-b) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/items", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-b","eval_case_id":"`+tenantBEval.ID+`"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestAddEvalDatasetItemEndpointRejectsPublishedDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, created.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-dataset",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/items", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset","eval_case_id":"`+evalCase.ID+`"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestPublishEvalDatasetEndpointPublishesDraft(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/publish", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset","published_by":"operator-publish"}`))
	if err != nil {
		t.Fatalf("Post(publish) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var published evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&published); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if published.Status != evalsvc.DatasetStatusPublished {
		t.Fatalf("Status = %q, want %q", published.Status, evalsvc.DatasetStatusPublished)
	}
	if published.PublishedBy != "operator-publish" {
		t.Fatalf("PublishedBy = %q, want %q", published.PublishedBy, "operator-publish")
	}
	if published.PublishedAt == "" {
		t.Fatal("published_at is empty")
	}
}

func TestPublishEvalDatasetEndpointRejectsRepublish(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Dataset source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	created, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, created.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-dataset",
	}); err != nil {
		t.Fatalf("PublishDataset(first) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-datasets/"+created.ID+"/publish", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-dataset"}`))
	if err != nil {
		t.Fatalf("Post(republish) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

type stubEvalDatasetStore struct {
	byID map[string]evalsvc.EvalDataset
}

func (s *stubEvalDatasetStore) CreateDataset(_ context.Context, item evalsvc.EvalDataset) (evalsvc.EvalDataset, error) {
	if s.byID == nil {
		s.byID = make(map[string]evalsvc.EvalDataset)
	}
	s.byID[item.ID] = item
	return item, nil
}

func (s *stubEvalDatasetStore) GetDataset(_ context.Context, datasetID string) (evalsvc.EvalDataset, error) {
	item, ok := s.byID[datasetID]
	if !ok {
		return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
	}
	return item, nil
}

func (s *stubEvalDatasetStore) ListDatasets(_ context.Context, _ evalsvc.DatasetListFilter) (evalsvc.DatasetListPage, error) {
	return evalsvc.DatasetListPage{}, nil
}

func (s *stubEvalDatasetStore) AddDatasetItem(_ context.Context, datasetID string, item evalsvc.EvalDatasetItem, updatedAt time.Time) (evalsvc.EvalDataset, error) {
	dataset, ok := s.byID[datasetID]
	if !ok {
		return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
	}
	if dataset.Status != evalsvc.DatasetStatusDraft {
		return evalsvc.EvalDataset{}, evalsvc.ErrInvalidEvalDatasetState
	}
	for _, existing := range dataset.Items {
		if existing.EvalCaseID == item.EvalCaseID {
			return dataset, nil
		}
	}
	dataset.Items = append(dataset.Items, item)
	dataset.UpdatedAt = updatedAt
	s.byID[datasetID] = dataset
	return dataset, nil
}

func (s *stubEvalDatasetStore) PublishDataset(_ context.Context, datasetID string, publishedBy string, publishedAt time.Time) (evalsvc.EvalDataset, error) {
	dataset, ok := s.byID[datasetID]
	if !ok {
		return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
	}
	if dataset.Status != evalsvc.DatasetStatusDraft {
		return evalsvc.EvalDataset{}, evalsvc.ErrInvalidEvalDatasetState
	}
	dataset.Status = evalsvc.DatasetStatusPublished
	dataset.PublishedBy = publishedBy
	dataset.PublishedAt = publishedAt
	dataset.UpdatedAt = publishedAt
	s.byID[datasetID] = dataset
	return dataset, nil
}

var _ interface {
	CreateDataset(context.Context, evalsvc.EvalDataset) (evalsvc.EvalDataset, error)
	GetDataset(context.Context, string) (evalsvc.EvalDataset, error)
	ListDatasets(context.Context, evalsvc.DatasetListFilter) (evalsvc.DatasetListPage, error)
	AddDatasetItem(context.Context, string, evalsvc.EvalDatasetItem, time.Time) (evalsvc.EvalDataset, error)
	PublishDataset(context.Context, string, string, time.Time) (evalsvc.EvalDataset, error)
} = (*stubEvalDatasetStore)(nil)
