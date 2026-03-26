package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
)

func TestCreateAndGetEvalRunEndpoint(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-run",
		Name:        "Published baseline",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-run",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-runs", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-run","dataset_id":"`+dataset.ID+`","created_by":"operator-run"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created evalRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if created.RunID == "" {
		t.Fatal("run_id is empty")
	}
	if created.Status != evalsvc.RunStatusQueued {
		t.Fatalf("Status = %q, want %q", created.Status, evalsvc.RunStatusQueued)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-runs/" + created.RunID + "?tenant_id=tenant-run")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}
}

func TestCreateEvalRunEndpointRejectsDraftDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-run",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-runs", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-run","dataset_id":"`+dataset.ID+`"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestCreateEvalRunEndpointRejectsCrossTenantDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-owner",
		Title:    "Run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-owner",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-owner",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-owner",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-runs", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-other","dataset_id":"`+dataset.ID+`"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestListEvalRunsEndpointSupportsFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	firstCase, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "A"})
	firstEval, _, _ := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{TenantID: "tenant-run", SourceCaseID: firstCase.ID})
	secondCase, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "B"})
	secondEval, _, _ := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{TenantID: "tenant-run", SourceCaseID: secondCase.ID})
	firstDataset, _ := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{TenantID: "tenant-run", Name: "A", EvalCaseIDs: []string{firstEval.ID}})
	secondDataset, _ := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{TenantID: "tenant-run", Name: "B", EvalCaseIDs: []string{secondEval.ID}})
	_, _ = datasetService.PublishDataset(ctx, firstDataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-run"})
	_, _ = datasetService.PublishDataset(ctx, secondDataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-run"})
	secondRun, _ := runService.CreateRun(ctx, evalsvc.CreateRunInput{TenantID: "tenant-run", DatasetID: secondDataset.ID})
	_, _ = runService.CreateRun(ctx, evalsvc.CreateRunInput{TenantID: "tenant-run", DatasetID: firstDataset.ID})

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run&dataset_id=" + secondDataset.ID + "&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listEvalRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].RunID != secondRun.ID {
		t.Fatalf("Runs = %#v, want only %q", page.Runs, secondRun.ID)
	}
}

func TestGetEvalRunEndpointRejectsWrongTenant(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-run",
		Name:        "Published baseline",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-run",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}
	run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-run",
		DatasetID: dataset.ID,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs/" + run.ID + "?tenant_id=tenant-other")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestListEvalRunsEndpointRejectsInvalidStatus(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        casesvc.NewService(),
		EvalCases:    evalsvc.NewService(casesvc.NewService(), nil),
		EvalDatasets: evalsvc.NewDatasetService(evalsvc.NewService(casesvc.NewService(), nil)),
		EvalRuns:     evalsvc.NewRunService(evalsvc.NewDatasetService(evalsvc.NewService(casesvc.NewService(), nil))),
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run&status=bogus")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetEvalRunEndpointReturnsUpdatedStatusFields(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-run",
		Name:        "Published baseline",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
		TenantID: "tenant-run",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-run",
		DatasetID: dataset.ID,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	claimed, err := runService.ClaimQueuedRuns(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(claimed) = %d, want 1", len(claimed))
	}
	if _, err := runService.MarkRunFailed(ctx, run.ID, "fault injection: eval run failed"); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs/" + run.ID + "?tenant_id=tenant-run")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got evalRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Status != evalsvc.RunStatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, evalsvc.RunStatusFailed)
	}
	if got.ErrorReason == "" {
		t.Fatal("ErrorReason is empty")
	}
	if got.StartedAt == "" {
		t.Fatal("StartedAt is empty")
	}
	if got.FinishedAt == "" {
		t.Fatal("FinishedAt is empty")
	}
}
