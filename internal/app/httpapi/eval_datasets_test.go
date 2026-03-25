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
