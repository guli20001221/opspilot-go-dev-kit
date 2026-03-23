package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"opspilot-go/internal/workflow"
)

// Store persists report read models.
type Store interface {
	Save(ctx context.Context, item Report) (Report, error)
	Get(ctx context.Context, reportID string) (Report, error)
}

type taskReportFinalizingStore interface {
	FinalizeSucceededTaskWithReport(ctx context.Context, task workflow.Task, event workflow.AuditEvent, item Report) (Report, workflow.Task, error)
}

// Service manages durable report read models.
type Service struct {
	store     Store
	finalizer taskReportFinalizingStore
}

// NewService constructs the report service with a memory-backed default store.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore constructs the report service with a caller-provided store.
func NewServiceWithStore(store Store) *Service {
	if store == nil {
		store = newMemoryStore()
	}

	service := &Service{store: store}
	if finalizer, ok := store.(taskReportFinalizingStore); ok {
		service.finalizer = finalizer
	}

	return service
}

// GetReport returns a report by ID.
func (s *Service) GetReport(ctx context.Context, reportID string) (Report, error) {
	return s.store.Get(ctx, reportID)
}

// RecordGeneratedReport persists the durable report emitted by a successful task.
func (s *Service) RecordGeneratedReport(ctx context.Context, task workflow.Task, result workflow.ExecutionResult) (string, error) {
	record := buildGeneratedReport(task, result)
	saved, err := s.store.Save(ctx, record)
	if err != nil {
		return "", err
	}

	return saved.ID, nil
}

// FinalizeGeneratedReportTask atomically persists the report and successful task state
// when the underlying store supports combined task/report transactions.
func (s *Service) FinalizeGeneratedReportTask(ctx context.Context, task workflow.Task, result workflow.ExecutionResult, event workflow.AuditEvent) (workflow.Task, string, error) {
	record := buildGeneratedReport(task, result)
	if s.finalizer == nil {
		return workflow.Task{}, "", fmt.Errorf("report store does not support atomic task finalization")
	}

	saved, updated, err := s.finalizer.FinalizeSucceededTaskWithReport(ctx, task, event, record)
	if err != nil {
		return workflow.Task{}, "", err
	}

	return updated, saved.ID, nil
}

// SupportsAtomicFinalization reports whether the underlying store can finalize
// task success and report persistence in one combined write path.
func (s *Service) SupportsAtomicFinalization() bool {
	return s.finalizer != nil
}

// ReportIDFromTaskID derives the stable report ID for a workflow task.
func ReportIDFromTaskID(taskID string) string {
	return "report-" + taskID
}

func buildMetadata(task workflow.Task, result workflow.ExecutionResult, readyAt time.Time) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
		"task_id":           task.ID,
		"request_id":        task.RequestID,
		"session_id":        task.SessionID,
		"task_type":         task.TaskType,
		"reason":            task.Reason,
		"audit_ref":         task.AuditRef,
		"execution_summary": result.Detail,
		"ready_at":          readyAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return payload
}

func buildGeneratedReport(task workflow.Task, result workflow.ExecutionResult) Report {
	now := task.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	readyAt := now
	record := Report{
		ID:           ReportIDFromTaskID(task.ID),
		TenantID:     task.TenantID,
		SourceTaskID: task.ID,
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        fmt.Sprintf("Report for %s", task.ID),
		Summary:      fallbackString(result.Detail, fmt.Sprintf("Generated report from task %s", task.ID)),
		ContentURI:   "",
		MetadataJSON: buildMetadata(task, result, now),
		CreatedBy:    "worker",
		CreatedAt:    task.CreatedAt,
		ReadyAt:      &readyAt,
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}

	return record
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}
