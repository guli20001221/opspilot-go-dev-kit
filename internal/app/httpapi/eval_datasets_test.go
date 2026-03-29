package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
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

	var created evalDatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if created.DatasetID == "" {
		t.Fatal("dataset_id is empty")
	}
	if len(created.Items) != 1 || created.Items[0].EvalCaseID != evalCase.ID {
		t.Fatalf("Items = %#v, want one eval case item", created.Items)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-datasets/" + created.DatasetID + "?tenant_id=tenant-dataset")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
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
