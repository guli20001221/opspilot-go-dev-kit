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
