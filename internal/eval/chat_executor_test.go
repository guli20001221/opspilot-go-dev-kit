package eval

import (
	"context"
	"fmt"
	"testing"

	appchat "opspilot-go/internal/app/chat"
	casesvc "opspilot-go/internal/case"
)

type mockChatService struct {
	calls []appchat.ChatRequestEnvelope
}

func (m *mockChatService) Handle(_ context.Context, req appchat.ChatRequestEnvelope) (appchat.HandleResult, error) {
	m.calls = append(m.calls, req)
	return appchat.HandleResult{SessionID: "mock-session"}, nil
}

func TestChatRunExecutorCallsChatForEachItem(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalCaseService)
	runService := NewRunService(datasetService)

	// Create source cases and promote to eval
	case1, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-exec",
		Title:    "How do I reset my password?",
	})
	if err != nil {
		t.Fatalf("CreateCase(1) error = %v", err)
	}
	case2, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-exec",
		Title:    "What is the refund policy?",
	})
	if err != nil {
		t.Fatalf("CreateCase(2) error = %v", err)
	}

	eval1, _, err := evalCaseService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-exec",
		SourceCaseID: case1.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(1) error = %v", err)
	}
	eval2, _, err := evalCaseService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-exec",
		SourceCaseID: case2.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(2) error = %v", err)
	}

	// Create dataset and publish
	dataset, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-exec",
		Name:        "Exec test dataset",
		EvalCaseIDs: []string{eval1.ID, eval2.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, PublishDatasetInput{
		TenantID: "tenant-exec",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	// Create and claim run
	run, err := runService.CreateRun(ctx, CreateRunInput{
		TenantID:  "tenant-exec",
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

	// Execute with chat executor
	chat := &mockChatService{}
	executor := NewChatRunExecutor(chat, runService)

	if err := executor.ExecuteRun(ctx, run); err != nil {
		t.Fatalf("ExecuteRun() error = %v", err)
	}

	// Verify chat was called for each eval case
	if len(chat.calls) != 2 {
		t.Fatalf("len(chat.calls) = %d, want 2", len(chat.calls))
	}
	if chat.calls[0].TenantID != "tenant-exec" {
		t.Fatalf("calls[0].TenantID = %q, want %q", chat.calls[0].TenantID, "tenant-exec")
	}
	if chat.calls[0].Mode != "eval" {
		t.Fatalf("calls[0].Mode = %q, want %q", chat.calls[0].Mode, "eval")
	}
}

type failingChatService struct {
	failOn string
	calls  []appchat.ChatRequestEnvelope
}

func (m *failingChatService) Handle(_ context.Context, req appchat.ChatRequestEnvelope) (appchat.HandleResult, error) {
	m.calls = append(m.calls, req)
	if m.failOn != "" && req.UserMessage == m.failOn {
		return appchat.HandleResult{}, fmt.Errorf("chat failed for: %s", req.UserMessage)
	}
	return appchat.HandleResult{SessionID: "mock-session"}, nil
}

func TestChatRunExecutorContinuesOnItemFailure(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalCaseService)
	runService := NewRunService(datasetService)

	case1, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-fail", Title: "Good case"})
	case2, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-fail", Title: "Bad case"})
	eval1, _, _ := evalCaseService.PromoteCase(ctx, CreateInput{TenantID: "tenant-fail", SourceCaseID: case1.ID})
	eval2, _, _ := evalCaseService.PromoteCase(ctx, CreateInput{TenantID: "tenant-fail", SourceCaseID: case2.ID})

	dataset, _ := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID: "tenant-fail", Name: "Fail dataset", EvalCaseIDs: []string{eval1.ID, eval2.ID},
	})
	datasetService.PublishDataset(ctx, dataset.ID, PublishDatasetInput{TenantID: "tenant-fail"})

	run, _ := runService.CreateRun(ctx, CreateRunInput{TenantID: "tenant-fail", DatasetID: dataset.ID})
	runService.ClaimQueuedRuns(ctx, 10)

	chat := &failingChatService{failOn: "Bad case"}
	executor := NewChatRunExecutor(chat, runService)

	err := executor.ExecuteRun(ctx, run)
	if err != nil {
		t.Fatalf("ExecuteRun() error = %v, want nil (individual failures logged, not propagated)", err)
	}
	if len(chat.calls) != 2 {
		t.Fatalf("len(chat.calls) = %d, want 2 (both items should be attempted)", len(chat.calls))
	}
}
