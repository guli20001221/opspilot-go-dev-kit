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
	if len(got.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 on detail response", len(got.Items))
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
	if got.ItemResults[0].EvalCaseID != evalCase.ID {
		t.Fatalf("ItemResults[0].EvalCaseID = %q, want %q", got.ItemResults[0].EvalCaseID, evalCase.ID)
	}
	if got.ItemResults[0].Status != evalsvc.RunItemResultFailed {
		t.Fatalf("ItemResults[0].Status = %q, want %q", got.ItemResults[0].Status, evalsvc.RunItemResultFailed)
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
