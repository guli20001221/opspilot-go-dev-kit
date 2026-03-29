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
	if got.Items[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.Items[0].PreferredFollowUpAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("Items[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", got.Items[0].PreferredFollowUpAction.SourceEvalCaseID, evalCase.ID)
	}
	if got.Items[0].PreferredLinkedCaseAction.Mode != "none" {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.Items[0].PreferredLinkedCaseAction.Mode, "none")
	}
	if got.Items[0].PreferredLinkedCaseAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.SourceEvalCaseID = %q, want %q", got.Items[0].PreferredLinkedCaseAction.SourceEvalCaseID, evalCase.ID)
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

func TestListEvalRunsEndpointIncludesFollowUpSummary(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	makeFailedRun := func(title string, withOpenFollowUp bool) (evalsvc.EvalRun, casesvc.Case) {
		t.Helper()
		sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
			TenantID: "tenant-run",
			Title:    title,
		})
		if err != nil {
			t.Fatalf("CreateCase(%q) error = %v", title, err)
		}
		evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
			TenantID:     "tenant-run",
			SourceCaseID: sourceCase.ID,
		})
		if err != nil {
			t.Fatalf("PromoteCase(%q) error = %v", title, err)
		}
		dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
			TenantID:    "tenant-run",
			Name:        title + " dataset",
			EvalCaseIDs: []string{evalCase.ID},
		})
		if err != nil {
			t.Fatalf("CreateDataset(%q) error = %v", title, err)
		}
		if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
			TenantID: "tenant-run",
		}); err != nil {
			t.Fatalf("PublishDataset(%q) error = %v", title, err)
		}
		run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
			TenantID:  "tenant-run",
			DatasetID: dataset.ID,
		})
		if err != nil {
			t.Fatalf("CreateRun(%q) error = %v", title, err)
		}
		if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
			t.Fatalf("ClaimQueuedRuns(%q) error = %v", title, err)
		}
		if _, err := runService.MarkRunFailed(ctx, run.ID, "fault injection: eval run failed"); err != nil {
			t.Fatalf("MarkRunFailed(%q) error = %v", title, err)
		}
		var followUpCase casesvc.Case
		if withOpenFollowUp {
			followUpCase, err = caseService.CreateCase(ctx, casesvc.CreateInput{
				TenantID:         "tenant-run",
				Title:            "Follow-up for " + title,
				Summary:          "Existing open follow-up",
				SourceEvalCaseID: evalCase.ID,
				SourceEvalRunID:  run.ID,
				CreatedBy:        "operator-run",
			})
			if err != nil {
				t.Fatalf("CreateCase(follow-up %q) error = %v", title, err)
			}
			followUpCase, err = caseService.AssignCase(ctx, followUpCase, "run-operator")
			if err != nil {
				t.Fatalf("AssignCase(follow-up %q) error = %v", title, err)
			}
		}
		return run, followUpCase
	}

	uncoveredRun, _ := makeFailedRun("Needs follow-up", false)
	coveredRun, coveredFollowUpCase := makeFailedRun("Already covered", true)

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

	var page listEvalRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode(page) error = %v", err)
	}
	if len(page.Runs) != 2 {
		t.Fatalf("len(page.Runs) = %d, want 2", len(page.Runs))
	}

	byID := make(map[string]evalRunResponse, len(page.Runs))
	for _, run := range page.Runs {
		byID[run.RunID] = run
	}
	if got := byID[uncoveredRun.ID].ItemWithoutOpenFollowUpCount; got != 1 {
		t.Fatalf("uncovered ItemWithoutOpenFollowUpCount = %d, want 1", got)
	}
	if !byID[uncoveredRun.ID].NeedsFollowUp {
		t.Fatal("uncovered NeedsFollowUp = false, want true")
	}
	if got := byID[coveredRun.ID].ItemWithoutOpenFollowUpCount; got != 0 {
		t.Fatalf("covered ItemWithoutOpenFollowUpCount = %d, want 0", got)
	}
	if byID[coveredRun.ID].NeedsFollowUp {
		t.Fatal("covered NeedsFollowUp = true, want false")
	}
	if got := byID[coveredRun.ID].LinkedCaseSummary.TotalCaseCount; got != 1 {
		t.Fatalf("covered LinkedCaseSummary.TotalCaseCount = %d, want 1", got)
	}
	if got := byID[coveredRun.ID].LinkedCaseSummary.OpenCaseCount; got != 1 {
		t.Fatalf("covered LinkedCaseSummary.OpenCaseCount = %d, want 1", got)
	}
	if got := byID[coveredRun.ID].LinkedCaseSummary.LatestCaseID; got != coveredFollowUpCase.ID {
		t.Fatalf("covered LinkedCaseSummary.LatestCaseID = %q, want %q", got, coveredFollowUpCase.ID)
	}
	if got := byID[coveredRun.ID].LinkedCaseSummary.LatestCaseStatus; got != casesvc.StatusOpen {
		t.Fatalf("covered LinkedCaseSummary.LatestCaseStatus = %q, want %q", got, casesvc.StatusOpen)
	}
	if got := byID[coveredRun.ID].LinkedCaseSummary.LatestAssignedTo; got != "run-operator" {
		t.Fatalf("covered LinkedCaseSummary.LatestAssignedTo = %q, want %q", got, "run-operator")
	}
	if got := byID[coveredRun.ID].PreferredLinkedCaseAction.Mode; got != "open_existing_case" {
		t.Fatalf("covered PreferredLinkedCaseAction.Mode = %q, want %q", got, "open_existing_case")
	}
	if got := byID[coveredRun.ID].PreferredLinkedCaseAction.CaseID; got != coveredFollowUpCase.ID {
		t.Fatalf("covered PreferredLinkedCaseAction.CaseID = %q, want %q", got, coveredFollowUpCase.ID)
	}
}

func TestListEvalRunsEndpointSupportsNeedsFollowUpFilter(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	makeFailedRun := func(title string, withOpenFollowUp bool) evalsvc.EvalRun {
		t.Helper()
		sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
			TenantID: "tenant-run",
			Title:    title,
		})
		if err != nil {
			t.Fatalf("CreateCase(%q) error = %v", title, err)
		}
		evalCase, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
			TenantID:     "tenant-run",
			SourceCaseID: sourceCase.ID,
		})
		if err != nil {
			t.Fatalf("PromoteCase(%q) error = %v", title, err)
		}
		dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
			TenantID:    "tenant-run",
			Name:        title + " dataset",
			EvalCaseIDs: []string{evalCase.ID},
		})
		if err != nil {
			t.Fatalf("CreateDataset(%q) error = %v", title, err)
		}
		if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{
			TenantID: "tenant-run",
		}); err != nil {
			t.Fatalf("PublishDataset(%q) error = %v", title, err)
		}
		run, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
			TenantID:  "tenant-run",
			DatasetID: dataset.ID,
		})
		if err != nil {
			t.Fatalf("CreateRun(%q) error = %v", title, err)
		}
		if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
			t.Fatalf("ClaimQueuedRuns(%q) error = %v", title, err)
		}
		if _, err := runService.MarkRunFailed(ctx, run.ID, "fault injection: eval run failed"); err != nil {
			t.Fatalf("MarkRunFailed(%q) error = %v", title, err)
		}
		if withOpenFollowUp {
			if _, err := caseService.CreateCase(ctx, casesvc.CreateInput{
				TenantID:         "tenant-run",
				Title:            "Follow-up for " + title,
				Summary:          "Existing open follow-up",
				SourceEvalCaseID: evalCase.ID,
				CreatedBy:        "operator-run",
			}); err != nil {
				t.Fatalf("CreateCase(follow-up %q) error = %v", title, err)
			}
		}
		return run
	}

	uncoveredRun := makeFailedRun("Needs follow-up", false)
	coveredRun := makeFailedRun("Already covered", true)

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
	}))
	defer server.Close()

	assertRunFilter := func(rawValue string, wantRunID string) {
		t.Helper()
		resp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run&status=failed&needs_follow_up=" + rawValue + "&limit=10")
		if err != nil {
			t.Fatalf("Get(%s) error = %v", rawValue, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("StatusCode(%s) = %d, want %d", rawValue, resp.StatusCode, http.StatusOK)
		}
		var page listEvalRunsResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			t.Fatalf("Decode(%s) error = %v", rawValue, err)
		}
		if len(page.Runs) != 1 {
			t.Fatalf("len(page.Runs) for %s = %d, want 1", rawValue, len(page.Runs))
		}
		if page.Runs[0].RunID != wantRunID {
			t.Fatalf("page.Runs[0].RunID for %s = %q, want %q", rawValue, page.Runs[0].RunID, wantRunID)
		}
	}

	assertRunFilter("true", uncoveredRun.ID)
	assertRunFilter("false", coveredRun.ID)
}

func TestListEvalRunsEndpointRejectsInvalidNeedsFollowUp(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        casesvc.NewService(),
		EvalCases:    evalsvc.NewService(casesvc.NewService(), nil),
		EvalDatasets: evalsvc.NewDatasetService(evalsvc.NewService(casesvc.NewService(), nil)),
		EvalRuns:     evalsvc.NewRunService(evalsvc.NewDatasetService(evalsvc.NewService(casesvc.NewService(), nil))),
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run&needs_follow_up=bogus")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
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
	if got.Items[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.Items[0].PreferredFollowUpAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("Items[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", got.Items[0].PreferredFollowUpAction.SourceEvalCaseID, evalCase.ID)
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
	if got.ItemResults[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("ItemResults[0].PreferredFollowUpAction.Mode = %q, want %q", got.ItemResults[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.ItemResults[0].PreferredFollowUpAction.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("ItemResults[0].PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", got.ItemResults[0].PreferredFollowUpAction.SourceEvalCaseID, evalCase.ID)
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

func TestGetEvalRunEndpointIncludesFollowUpReuseActions(t *testing.T) {
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
	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:         "tenant-run",
		Title:            "Eval follow-up",
		Summary:          "Existing eval follow-up",
		SourceEvalCaseID: evalCase.ID,
		SourceEvalRunID:  run.ID,
		CreatedBy:        "operator-run",
	})
	if err != nil {
		t.Fatalf("CreateCase(followUpCase) error = %v", err)
	}
	followUpCase, err = caseService.AssignCase(ctx, followUpCase, "detail-run-operator")
	if err != nil {
		t.Fatalf("AssignCase(followUpCase) error = %v", err)
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
	if got.Items[0].LatestFollowUpCaseID != followUpCase.ID {
		t.Fatalf("Items[0].LatestFollowUpCaseID = %q, want %q", got.Items[0].LatestFollowUpCaseID, followUpCase.ID)
	}
	if got.Items[0].LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("Items[0].LinkedCaseSummary.TotalCaseCount = %d, want 1", got.Items[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("Items[0].LinkedCaseSummary.OpenCaseCount = %d, want 1", got.Items[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("Items[0].LinkedCaseSummary.LatestCaseID = %q, want %q", got.Items[0].LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.Items[0].LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("Items[0].LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.Items[0].LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.Items[0].PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if got.Items[0].PreferredFollowUpAction.CaseID != followUpCase.ID {
		t.Fatalf("Items[0].PreferredFollowUpAction.CaseID = %q, want %q", got.Items[0].PreferredFollowUpAction.CaseID, followUpCase.ID)
	}
	if got.Items[0].PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.Items[0].PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.Items[0].PreferredLinkedCaseAction.CaseID != followUpCase.ID {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.CaseID = %q, want %q", got.Items[0].PreferredLinkedCaseAction.CaseID, followUpCase.ID)
	}
	if got.ItemResults[0].LatestFollowUpCaseID != followUpCase.ID {
		t.Fatalf("ItemResults[0].LatestFollowUpCaseID = %q, want %q", got.ItemResults[0].LatestFollowUpCaseID, followUpCase.ID)
	}
	if got.ItemResults[0].LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("ItemResults[0].LinkedCaseSummary.TotalCaseCount = %d, want 1", got.ItemResults[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.ItemResults[0].LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("ItemResults[0].LinkedCaseSummary.OpenCaseCount = %d, want 1", got.ItemResults[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.ItemResults[0].LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("ItemResults[0].LinkedCaseSummary.LatestCaseID = %q, want %q", got.ItemResults[0].LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.ItemResults[0].LinkedCaseSummary.LatestCaseStatus != casesvc.StatusOpen {
		t.Fatalf("ItemResults[0].LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.ItemResults[0].LinkedCaseSummary.LatestCaseStatus, casesvc.StatusOpen)
	}
	if got.ItemResults[0].PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("ItemResults[0].PreferredFollowUpAction.Mode = %q, want %q", got.ItemResults[0].PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if got.ItemResults[0].PreferredFollowUpAction.CaseID != followUpCase.ID {
		t.Fatalf("ItemResults[0].PreferredFollowUpAction.CaseID = %q, want %q", got.ItemResults[0].PreferredFollowUpAction.CaseID, followUpCase.ID)
	}
	if got.ItemResults[0].PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("ItemResults[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.ItemResults[0].PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.ItemResults[0].PreferredLinkedCaseAction.CaseID != followUpCase.ID {
		t.Fatalf("ItemResults[0].PreferredLinkedCaseAction.CaseID = %q, want %q", got.ItemResults[0].PreferredLinkedCaseAction.CaseID, followUpCase.ID)
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
	if got.LinkedCaseSummary.LatestAssignedTo != "detail-run-operator" {
		t.Fatalf("LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "detail-run-operator")
	}
	if got.PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredLinkedCaseAction.CaseID != followUpCase.ID {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want %q", got.PreferredLinkedCaseAction.CaseID, followUpCase.ID)
	}
}

func TestGetEvalRunEndpointSuppressesDirectLinkedCaseActionForClosedLatestCase(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run source closed latest case",
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
		Name:        "Closed latest case baseline",
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
	followUpCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:         "tenant-run",
		Title:            "Closed eval follow-up",
		Summary:          "Closed follow-up",
		SourceEvalCaseID: evalCase.ID,
		SourceEvalRunID:  run.ID,
		CreatedBy:        "operator-run",
	})
	if err != nil {
		t.Fatalf("CreateCase(followUpCase) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, followUpCase.ID, "detail-run-operator"); err != nil {
		t.Fatalf("CloseCase(followUpCase) error = %v", err)
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
	if got.LinkedCaseSummary.LatestCaseID != followUpCase.ID {
		t.Fatalf("LinkedCaseSummary.LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, followUpCase.ID)
	}
	if got.LinkedCaseSummary.LatestCaseStatus != casesvc.StatusClosed {
		t.Fatalf("LinkedCaseSummary.LatestCaseStatus = %q, want %q", got.LinkedCaseSummary.LatestCaseStatus, casesvc.StatusClosed)
	}
	if got.PreferredLinkedCaseAction.Mode != "none" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "none")
	}
	if got.PreferredLinkedCaseAction.CaseID != "" {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want empty", got.PreferredLinkedCaseAction.CaseID)
	}
	if got.Items[0].PreferredLinkedCaseAction.Mode != "open_existing_queue" {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.Items[0].PreferredLinkedCaseAction.Mode, "open_existing_queue")
	}
	if got.Items[0].PreferredLinkedCaseAction.CaseID != "" {
		t.Fatalf("Items[0].PreferredLinkedCaseAction.CaseID = %q, want empty", got.Items[0].PreferredLinkedCaseAction.CaseID)
	}
	if got.ItemResults[0].PreferredLinkedCaseAction.Mode != "open_existing_queue" {
		t.Fatalf("ItemResults[0].PreferredLinkedCaseAction.Mode = %q, want %q", got.ItemResults[0].PreferredLinkedCaseAction.Mode, "open_existing_queue")
	}
	if got.ItemResults[0].PreferredLinkedCaseAction.CaseID != "" {
		t.Fatalf("ItemResults[0].PreferredLinkedCaseAction.CaseID = %q, want empty", got.ItemResults[0].PreferredLinkedCaseAction.CaseID)
	}
}

func TestGetEvalRunEndpointUsesRunBackedLinkedCaseSummary(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Run-backed source",
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
		Name:        "Run-backed queue",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-run"}); err != nil {
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

	runBackedCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:        "tenant-run",
		Title:           "Run-backed only follow-up",
		Summary:         "Created from run queue",
		SourceEvalRunID: run.ID,
		CreatedBy:       "operator-run",
	})
	if err != nil {
		t.Fatalf("CreateCase(runBackedCase) error = %v", err)
	}
	runBackedCase, err = caseService.AssignCase(ctx, runBackedCase, "run-owner")
	if err != nil {
		t.Fatalf("AssignCase(runBackedCase) error = %v", err)
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
	if got.LinkedCaseSummary.TotalCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.TotalCaseCount = %d, want 1", got.LinkedCaseSummary.TotalCaseCount)
	}
	if got.LinkedCaseSummary.OpenCaseCount != 1 {
		t.Fatalf("LinkedCaseSummary.OpenCaseCount = %d, want 1", got.LinkedCaseSummary.OpenCaseCount)
	}
	if got.LinkedCaseSummary.LatestCaseID != runBackedCase.ID {
		t.Fatalf("LinkedCaseSummary.LatestCaseID = %q, want %q", got.LinkedCaseSummary.LatestCaseID, runBackedCase.ID)
	}
	if got.LinkedCaseSummary.LatestAssignedTo != "run-owner" {
		t.Fatalf("LinkedCaseSummary.LatestAssignedTo = %q, want %q", got.LinkedCaseSummary.LatestAssignedTo, "run-owner")
	}
	if got.PreferredLinkedCaseAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredLinkedCaseAction.Mode = %q, want %q", got.PreferredLinkedCaseAction.Mode, "open_existing_case")
	}
	if got.PreferredLinkedCaseAction.CaseID != runBackedCase.ID {
		t.Fatalf("PreferredLinkedCaseAction.CaseID = %q, want %q", got.PreferredLinkedCaseAction.CaseID, runBackedCase.ID)
	}
	if got.Items[0].PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("Items[0].PreferredFollowUpAction.Mode = %q, want %q", got.Items[0].PreferredFollowUpAction.Mode, "create")
	}
	if got.Items[0].LinkedCaseSummary.TotalCaseCount != 0 {
		t.Fatalf("Items[0].LinkedCaseSummary.TotalCaseCount = %d, want 0", got.Items[0].LinkedCaseSummary.TotalCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.OpenCaseCount != 0 {
		t.Fatalf("Items[0].LinkedCaseSummary.OpenCaseCount = %d, want 0", got.Items[0].LinkedCaseSummary.OpenCaseCount)
	}
	if got.Items[0].LinkedCaseSummary.LatestCaseID != "" {
		t.Fatalf("Items[0].LinkedCaseSummary.LatestCaseID = %q, want empty", got.Items[0].LinkedCaseSummary.LatestCaseID)
	}
}

func TestEvalRunEndpointsIncludeMaterializedReportLinkage(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	reportID := materializeEvalRunReport(t, "tenant-run-report-link", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Run Link", "Source Run Link")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:        caseService,
		EvalCases:    evalCaseService,
		EvalDatasets: datasetService,
		EvalRuns:     runService,
		EvalReports:  reportService,
	}))
	defer server.Close()

	listResp, err := http.Get(server.URL + "/api/v1/eval-runs?tenant_id=tenant-run-report-link&status=failed&limit=10")
	if err != nil {
		t.Fatalf("Get(list) error = %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list StatusCode = %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	var page listEvalRunsResponse
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode(list page) error = %v", err)
	}
	if len(page.Runs) != 1 {
		t.Fatalf("len(page.Runs) = %d, want 1", len(page.Runs))
	}
	if page.Runs[0].ReportID != reportID {
		t.Fatalf("page.Runs[0].ReportID = %q, want %q", page.Runs[0].ReportID, reportID)
	}
	if page.Runs[0].ReportStatus != evalsvc.EvalReportStatusReady {
		t.Fatalf("page.Runs[0].ReportStatus = %q, want %q", page.Runs[0].ReportStatus, evalsvc.EvalReportStatusReady)
	}

	detailResp, err := http.Get(server.URL + "/api/v1/eval-runs/" + page.Runs[0].RunID + "?tenant_id=tenant-run-report-link")
	if err != nil {
		t.Fatalf("Get(detail) error = %v", err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusOK {
		t.Fatalf("detail StatusCode = %d, want %d", detailResp.StatusCode, http.StatusOK)
	}

	var detail evalRunResponse
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatalf("Decode(detail) error = %v", err)
	}
	if detail.ReportID != reportID {
		t.Fatalf("detail.ReportID = %q, want %q", detail.ReportID, reportID)
	}
	if detail.ReportStatus != evalsvc.EvalReportStatusReady {
		t.Fatalf("detail.ReportStatus = %q, want %q", detail.ReportStatus, evalsvc.EvalReportStatusReady)
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
