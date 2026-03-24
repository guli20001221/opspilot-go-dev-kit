package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"opspilot-go/internal/workflow"
)

// Store persists report read models.
type Store interface {
	Save(ctx context.Context, item Report) (Report, error)
	Get(ctx context.Context, reportID string) (Report, error)
	List(ctx context.Context, filter ListFilter) (ListPage, error)
}

type taskReportFinalizingStore interface {
	FinalizeSucceededTaskWithReport(ctx context.Context, task workflow.Task, event workflow.AuditEvent, item Report) (Report, workflow.Task, error)
}

type currentVersionSource interface {
	CurrentVersionID(ctx context.Context) (string, error)
}

// Service manages durable report read models.
type Service struct {
	store          Store
	finalizer      taskReportFinalizingStore
	currentVersion currentVersionSource
}

// NewService constructs the report service with a memory-backed default store.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore constructs the report service with a caller-provided store.
func NewServiceWithStore(store Store) *Service {
	return NewServiceWithDependencies(store, nil)
}

// NewServiceWithDependencies constructs the report service with caller-provided dependencies.
func NewServiceWithDependencies(store Store, currentVersion currentVersionSource) *Service {
	if store == nil {
		store = newMemoryStore()
	}

	service := &Service{
		store:          store,
		currentVersion: currentVersion,
	}
	if finalizer, ok := store.(taskReportFinalizingStore); ok {
		service.finalizer = finalizer
	}

	return service
}

// GetReport returns a report by ID.
func (s *Service) GetReport(ctx context.Context, reportID string) (Report, error) {
	return s.store.Get(ctx, reportID)
}

// ListReports returns a durable report page.
func (s *Service) ListReports(ctx context.Context, filter ListFilter) (ListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return s.store.List(ctx, filter)
}

// CompareReports returns two durable reports plus an operator-facing summary
// of the differences that matter for triage and regression review.
func (s *Service) CompareReports(ctx context.Context, leftReportID string, rightReportID string) (Comparison, error) {
	if leftReportID == "" {
		return Comparison{}, errors.New("left_report_id is required")
	}
	if rightReportID == "" {
		return Comparison{}, errors.New("right_report_id is required")
	}

	left, err := s.store.Get(ctx, leftReportID)
	if err != nil {
		return Comparison{}, err
	}
	right, err := s.store.Get(ctx, rightReportID)
	if err != nil {
		return Comparison{}, err
	}

	return Comparison{
		Left:    left,
		Right:   right,
		Summary: buildComparisonSummary(left, right),
	}, nil
}

// RecordGeneratedReport persists the durable report emitted by a successful task.
func (s *Service) RecordGeneratedReport(ctx context.Context, task workflow.Task, result workflow.ExecutionResult) (string, error) {
	record, err := s.buildGeneratedReport(ctx, task, result)
	if err != nil {
		return "", err
	}
	saved, err := s.store.Save(ctx, record)
	if err != nil {
		return "", err
	}

	return saved.ID, nil
}

// FinalizeGeneratedReportTask atomically persists the report and successful task state
// when the underlying store supports combined task/report transactions.
func (s *Service) FinalizeGeneratedReportTask(ctx context.Context, task workflow.Task, result workflow.ExecutionResult, event workflow.AuditEvent) (workflow.Task, string, error) {
	record, err := s.buildGeneratedReport(ctx, task, result)
	if err != nil {
		return workflow.Task{}, "", err
	}
	if s.finalizer == nil {
		return workflow.Task{}, "", fmt.Errorf("report store does not support atomic task finalization")
	}
	if task.VersionID == "" && record.VersionID != "" {
		task.VersionID = record.VersionID
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

func buildMetadata(task workflow.Task, result workflow.ExecutionResult, readyAt time.Time, versionID string) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
		"task_id":           task.ID,
		"request_id":        task.RequestID,
		"session_id":        task.SessionID,
		"task_type":         task.TaskType,
		"version_id":        versionID,
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

func (s *Service) buildGeneratedReport(ctx context.Context, task workflow.Task, result workflow.ExecutionResult) (Report, error) {
	versionID := task.VersionID
	if versionID == "" && s.currentVersion != nil {
		currentVersionID, err := s.currentVersion.CurrentVersionID(ctx)
		if err != nil {
			return Report{}, err
		}
		versionID = currentVersionID
	}

	now := task.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	readyAt := now
	record := Report{
		ID:           ReportIDFromTaskID(task.ID),
		TenantID:     task.TenantID,
		SourceTaskID: task.ID,
		VersionID:    versionID,
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        fmt.Sprintf("Report for %s", task.ID),
		Summary:      fallbackString(result.Detail, fmt.Sprintf("Generated report from task %s", task.ID)),
		ContentURI:   "",
		MetadataJSON: buildMetadata(task, result, now, versionID),
		CreatedBy:    "worker",
		CreatedAt:    task.CreatedAt,
		ReadyAt:      &readyAt,
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}

	return record, nil
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

func buildComparisonSummary(left Report, right Report) ComparisonSummary {
	summary := ComparisonSummary{
		SameTenant:        left.TenantID == right.TenantID,
		SameReportType:    left.ReportType == right.ReportType,
		VersionChanged:    left.VersionID != right.VersionID,
		SourceTaskChanged: left.SourceTaskID != right.SourceTaskID,
		TitleChanged:      left.Title != right.Title,
		SummaryChanged:    left.Summary != right.Summary,
		ContentURIChanged: left.ContentURI != right.ContentURI,
		MetadataChanged:   !equalJSON(left.MetadataJSON, right.MetadataJSON),
		CreatedAtChanged:  !left.CreatedAt.Equal(right.CreatedAt),
		ReadyAtChanged:    !equalOptionalTimes(left.ReadyAt, right.ReadyAt),
	}
	summary.ReadyAtDeltaSecond = deltaSeconds(left.ReadyAt, right.ReadyAt)

	return summary
}

func equalOptionalTimes(left *time.Time, right *time.Time) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	return left.Equal(*right)
}

func deltaSeconds(left *time.Time, right *time.Time) int64 {
	if left == nil || right == nil {
		return 0
	}

	return int64(right.Sub(*left).Seconds())
}

func equalJSON(left json.RawMessage, right json.RawMessage) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}

	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return string(left) == string(right)
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return string(left) == string(right)
	}

	return reflect.DeepEqual(leftValue, rightValue)
}
