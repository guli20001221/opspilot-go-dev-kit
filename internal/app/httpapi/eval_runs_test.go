package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var created evalRunResponse
	if err := json.Unmarshal(bodyBytes, &created); err != nil {
		t.Fatalf("Unmarshal(created) error = %v", err)
	}
	var createdRaw map[string]any
	if err := json.Unmarshal(bodyBytes, &createdRaw); err != nil {
		t.Fatalf("Unmarshal(createdRaw) error = %v", err)
	}
	if created.RunID == "" {
		t.Fatal("run_id is empty")
	}
	if created.Status != evalsvc.RunStatusQueued {
		t.Fatalf("Status = %q, want %q", created.Status, evalsvc.RunStatusQueued)
	}
	if len(created.Events) != 0 {
		t.Fatalf("len(Events) = %d, want 0 on create response", len(created.Events))
	}
	if len(created.Items) != 0 {
		t.Fatalf("len(Items) = %d, want 0 on create response", len(created.Items))
	}
	if _, ok := createdRaw["events"]; ok {
		t.Fatalf("create response unexpectedly included events field: %#v", createdRaw)
	}
	if _, ok := createdRaw["items"]; ok {
		t.Fatalf("create response unexpectedly included items field: %#v", createdRaw)
	}
	if _, ok := createdRaw["item_results"]; ok {
		t.Fatalf("create response unexpectedly included item_results field: %#v", createdRaw)
	}
	if _, ok := createdRaw["result_summary"]; ok {
		t.Fatalf("create response unexpectedly included result_summary field: %#v", createdRaw)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-runs/" + created.RunID + "?tenant_id=tenant-run")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}
	getBodyBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("ReadAll(get detail) error = %v", err)
	}
	var got evalRunResponse
	if err := json.Unmarshal(getBodyBytes, &got); err != nil {
		t.Fatalf("Unmarshal(get detail) error = %v", err)
	}
	var detailRaw map[string]any
	if err := json.Unmarshal(getBodyBytes, &detailRaw); err != nil {
		t.Fatalf("Unmarshal(detailRaw) error = %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 on detail response", len(got.Items))
	}
	if _, ok := detailRaw["result_summary"]; ok {
		t.Fatalf("queued detail unexpectedly included result_summary field: %#v", detailRaw)
	}
	if got.Items[0].EvalCaseID != evalCase.ID {
		t.Fatalf("Items[0].EvalCaseID = %q, want %q", got.Items[0].EvalCaseID, evalCase.ID)
	}
	if got.Items[0].Title != "Run source" {
		t.Fatalf("Items[0].Title = %q, want %q", got.Items[0].Title, "Run source")
	}
	if got.Items[0].SourceCaseID != sourceCase.ID {
		t.Fatalf("Items[0].SourceCaseID = %q, want %q", got.Items[0].SourceCaseID, sourceCase.ID)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var page listEvalRunsResponse
	if err := json.Unmarshal(bodyBytes, &page); err != nil {
		t.Fatalf("Unmarshal(page) error = %v", err)
	}
	var pageRaw map[string]any
	if err := json.Unmarshal(bodyBytes, &pageRaw); err != nil {
		t.Fatalf("Unmarshal(pageRaw) error = %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].RunID != secondRun.ID {
		t.Fatalf("Runs = %#v, want only %q", page.Runs, secondRun.ID)
	}
	if len(page.Runs[0].Events) != 0 {
		t.Fatalf("len(Events) = %d, want 0 on list response", len(page.Runs[0].Events))
	}
	if len(page.Runs[0].Items) != 0 {
		t.Fatalf("len(Items) = %d, want 0 on list response", len(page.Runs[0].Items))
	}
	rawRuns, ok := pageRaw["runs"].([]any)
	if !ok || len(rawRuns) != 1 {
		t.Fatalf("raw runs = %#v, want one item", pageRaw["runs"])
	}
	rawItem, ok := rawRuns[0].(map[string]any)
	if !ok {
		t.Fatalf("raw item = %#v, want object", rawRuns[0])
	}
	if _, ok := rawItem["events"]; ok {
		t.Fatalf("list response unexpectedly included events field: %#v", rawItem)
	}
	if _, ok := rawItem["items"]; ok {
		t.Fatalf("list response unexpectedly included items field: %#v", rawItem)
	}
	if _, ok := rawItem["item_results"]; ok {
		t.Fatalf("list response unexpectedly included item_results field: %#v", rawItem)
	}
	if _, ok := rawItem["result_summary"]; ok {
		t.Fatalf("list response unexpectedly included result_summary field for queued run: %#v", rawItem)
	}
}

func TestListEvalRunsEndpointReturnsResultSummaryForTerminalRuns(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCaseA, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "A"})
	evalCaseA, _, _ := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{TenantID: "tenant-run", SourceCaseID: sourceCaseA.ID})
	sourceCaseB, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "B"})
	evalCaseB, _, _ := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{TenantID: "tenant-run", SourceCaseID: sourceCaseB.ID})
	dataset, _ := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-run",
		Name:        "Results dataset",
		EvalCaseIDs: []string{evalCaseA.ID, evalCaseB.ID},
	})
	_, _ = datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-run"})
	run, _ := runService.CreateRun(ctx, evalsvc.CreateRunInput{TenantID: "tenant-run", DatasetID: dataset.ID})
	_, _ = runService.ClaimQueuedRuns(ctx, 10)
	_, _ = runService.MarkRunFailed(ctx, run.ID, "fault injection")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run&status=failed&limit=10")
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
	var page listEvalRunsResponse
	if err := json.Unmarshal(bodyBytes, &page); err != nil {
		t.Fatalf("Unmarshal(page) error = %v", err)
	}
	if len(page.Runs) != 1 {
		t.Fatalf("len(page.Runs) = %d, want 1", len(page.Runs))
	}
	if page.Runs[0].ResultSummary == nil {
		t.Fatal("ResultSummary = nil, want counts on terminal run")
	}
	if page.Runs[0].ResultSummary.TotalItems != 2 {
		t.Fatalf("TotalItems = %d, want 2", page.Runs[0].ResultSummary.TotalItems)
	}
	if page.Runs[0].ResultSummary.RecordedResults != 2 || page.Runs[0].ResultSummary.MissingResults != 0 {
		t.Fatalf("ResultSummary = %#v, want fully recorded terminal results", page.Runs[0].ResultSummary)
	}
	if page.Runs[0].ResultSummary.FailedItems != 2 || page.Runs[0].ResultSummary.SucceededItems != 0 {
		t.Fatalf("ResultSummary = %#v, want two failures and zero successes", page.Runs[0].ResultSummary)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got evalRunResponse
	if err := json.Unmarshal(bodyBytes, &got); err != nil {
		t.Fatalf("Unmarshal(got) error = %v", err)
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
	if len(got.Events) != 3 {
		t.Fatalf("len(Events) = %d, want 3", len(got.Events))
	}
	if got.Events[0].Action != evalsvc.RunEventCreated || got.Events[1].Action != evalsvc.RunEventClaimed || got.Events[2].Action != evalsvc.RunEventFailed {
		t.Fatalf("events = %#v, want created/claimed/failed", got.Events)
	}
	if len(got.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 on detail response", len(got.Items))
	}
	if got.Items[0].EvalCaseID != evalCase.ID {
		t.Fatalf("Items[0].EvalCaseID = %q, want %q", got.Items[0].EvalCaseID, evalCase.ID)
	}
	if len(got.ItemResults) != 1 {
		t.Fatalf("len(ItemResults) = %d, want 1 on detail response", len(got.ItemResults))
	}
	if got.ResultSummary == nil {
		t.Fatal("ResultSummary = nil, want terminal counts on detail response")
	}
	if got.ResultSummary.TotalItems != 1 || got.ResultSummary.FailedItems != 1 || got.ResultSummary.SucceededItems != 0 {
		t.Fatalf("ResultSummary = %#v, want one failed item", got.ResultSummary)
	}
	if got.ResultSummary.RecordedResults != 1 || got.ResultSummary.MissingResults != 0 {
		t.Fatalf("ResultSummary = %#v, want exactly one recorded result", got.ResultSummary)
	}
	if got.ItemResults[0].EvalCaseID != evalCase.ID {
		t.Fatalf("ItemResults[0].EvalCaseID = %q, want %q", got.ItemResults[0].EvalCaseID, evalCase.ID)
	}
	if got.ItemResults[0].Status != evalsvc.RunItemResultFailed {
		t.Fatalf("ItemResults[0].Status = %q, want %q", got.ItemResults[0].Status, evalsvc.RunItemResultFailed)
	}
	if got.ItemResults[0].Verdict != "fail" {
		t.Fatalf("ItemResults[0].Verdict = %q, want %q", got.ItemResults[0].Verdict, "fail")
	}
	if got.ItemResults[0].Score != 0 {
		t.Fatalf("ItemResults[0].Score = %v, want 0", got.ItemResults[0].Score)
	}
	if got.ItemResults[0].JudgeVersion == "" {
		t.Fatal("ItemResults[0].JudgeVersion is empty")
	}
	if len(got.ItemResults[0].JudgeOutput) == 0 {
		t.Fatal("ItemResults[0].JudgeOutput is empty")
	}
	var detailRaw map[string]any
	if err := json.Unmarshal(bodyBytes, &detailRaw); err != nil {
		t.Fatalf("Unmarshal(detailRaw) error = %v", err)
	}
	itemResultsRaw, ok := detailRaw["item_results"].([]any)
	if !ok || len(itemResultsRaw) != 1 {
		t.Fatalf("detailRaw item_results = %#v, want one raw item result", detailRaw["item_results"])
	}
	itemResultRaw, ok := itemResultsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("itemResultsRaw[0] = %#v, want object", itemResultsRaw[0])
	}
	for _, field := range []string{"verdict", "score", "judge_version", "judge_output"} {
		if _, ok := itemResultRaw[field]; !ok {
			t.Fatalf("detail item result missing raw %q field: %#v", field, itemResultRaw)
		}
	}
}

func TestRetryEvalRunEndpointRequeuesFailedRun(t *testing.T) {
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
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
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

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/eval-runs/"+run.ID+"/retry?tenant_id=tenant-run", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got evalRunResponse
	if err := json.Unmarshal(bodyBytes, &got); err != nil {
		t.Fatalf("Unmarshal(got) error = %v", err)
	}
	var retryRaw map[string]any
	if err := json.Unmarshal(bodyBytes, &retryRaw); err != nil {
		t.Fatalf("Unmarshal(retryRaw) error = %v", err)
	}
	if got.Status != evalsvc.RunStatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, evalsvc.RunStatusQueued)
	}
	if got.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", got.ErrorReason)
	}
	if got.StartedAt != "" {
		t.Fatalf("StartedAt = %q, want empty", got.StartedAt)
	}
	if got.FinishedAt != "" {
		t.Fatalf("FinishedAt = %q, want empty", got.FinishedAt)
	}
	if len(got.Events) != 0 {
		t.Fatalf("len(Events) = %d, want 0 on retry response", len(got.Events))
	}
	if len(got.Items) != 0 {
		t.Fatalf("len(Items) = %d, want 0 on retry response", len(got.Items))
	}
	if len(got.ItemResults) != 0 {
		t.Fatalf("len(ItemResults) = %d, want 0 on retry response", len(got.ItemResults))
	}
	if _, ok := retryRaw["events"]; ok {
		t.Fatalf("retry response unexpectedly included events field: %#v", retryRaw)
	}
	if _, ok := retryRaw["items"]; ok {
		t.Fatalf("retry response unexpectedly included items field: %#v", retryRaw)
	}
	if _, ok := retryRaw["item_results"]; ok {
		t.Fatalf("retry response unexpectedly included item_results field: %#v", retryRaw)
	}
	if _, ok := retryRaw["result_summary"]; ok {
		t.Fatalf("retry response unexpectedly included result_summary field: %#v", retryRaw)
	}
}

func TestRetryEvalRunEndpointRejectsInvalidState(t *testing.T) {
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

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/eval-runs/"+run.ID+"/retry?tenant_id=tenant-run", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}
