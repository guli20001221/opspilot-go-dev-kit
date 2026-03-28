package cases

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var caseIDSequence atomic.Uint64
var caseNoteIDSequence atomic.Uint64

// Store persists case read models.
type Store interface {
	Save(ctx context.Context, item Case) (Case, error)
	Get(ctx context.Context, caseID string) (Case, error)
	List(ctx context.Context, filter ListFilter) (ListPage, error)
	FindOpenByCompareOrigin(ctx context.Context, tenantID string, sourceEvalReportID string, compareOrigin CompareOrigin) (Case, bool, error)
	SummarizeBySourceEvalReportIDs(ctx context.Context, tenantID string, reportIDs []string) (map[string]EvalReportFollowUpSummary, error)
	SummarizeCompareOriginBySourceEvalReportIDs(ctx context.Context, tenantID string, reportIDs []string) (map[string]EvalReportCompareFollowUpSummary, error)
	SummarizeBySourceEvalCaseIDs(ctx context.Context, tenantID string, evalCaseIDs []string) (map[string]EvalCaseFollowUpSummary, error)
	AppendNote(ctx context.Context, note Note) (Note, error)
	ListNotes(ctx context.Context, caseID string, limit int) ([]Note, error)
	Assign(ctx context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (Case, error)
	Unassign(ctx context.Context, caseID string, unassignedBy string, unassignedAt time.Time, expectedUpdatedAt time.Time) (Case, error)
	Close(ctx context.Context, caseID string, closedBy string, closedAt time.Time) (Case, error)
	Reopen(ctx context.Context, caseID string, reopenedBy string, reopenedAt time.Time) (Case, error)
}

// Service manages durable operator case records.
type Service struct {
	store Store
}

// NewService constructs the case service with a memory-backed default store.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore constructs the case service with a caller-provided store.
func NewServiceWithStore(store Store) *Service {
	if store == nil {
		store = newMemoryStore()
	}

	return &Service{store: store}
}

// CreateCase persists a new operator-managed case.
func (s *Service) CreateCase(ctx context.Context, input CreateInput) (Case, error) {
	now := time.Now().UTC()
	item := Case{
		ID:                 newCaseID(now),
		TenantID:           input.TenantID,
		Status:             StatusOpen,
		Title:              input.Title,
		Summary:            input.Summary,
		SourceTaskID:       input.SourceTaskID,
		SourceReportID:     input.SourceReportID,
		SourceEvalReportID: input.SourceEvalReportID,
		SourceEvalCaseID:   input.SourceEvalCaseID,
		CompareOrigin:      input.CompareOrigin,
		CreatedBy:          fallbackString(input.CreatedBy, "operator"),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return s.store.Save(ctx, item)
}

// GetCase returns a durable case by ID.
func (s *Service) GetCase(ctx context.Context, caseID string) (Case, error) {
	return s.store.Get(ctx, caseID)
}

// ListCases returns operator-facing case rows for the provided filter.
func (s *Service) ListCases(ctx context.Context, filter ListFilter) (ListPage, error) {
	return s.store.List(ctx, filter)
}

// FindOpenCaseBySourceEvalCase returns the newest open case for one source eval case when it exists.
func (s *Service) FindOpenCaseBySourceEvalCase(ctx context.Context, tenantID string, sourceEvalCaseID string) (Case, bool, error) {
	page, err := s.store.List(ctx, ListFilter{
		TenantID:             tenantID,
		Status:               StatusOpen,
		ExcludeCompareOrigin: true,
		SourceEvalCaseID:     sourceEvalCaseID,
		Limit:                1,
	})
	if err != nil {
		return Case{}, false, err
	}
	if len(page.Cases) == 0 {
		return Case{}, false, nil
	}

	return page.Cases[0], true, nil
}

// FindOpenCaseBySourceEvalReport returns the newest open case for one source eval report when it exists.
func (s *Service) FindOpenCaseBySourceEvalReport(ctx context.Context, tenantID string, sourceEvalReportID string) (Case, bool, error) {
	page, err := s.store.List(ctx, ListFilter{
		TenantID:             tenantID,
		Status:               StatusOpen,
		ExcludeCompareOrigin: true,
		SourceEvalReportID:   sourceEvalReportID,
		Limit:                1,
	})
	if err != nil {
		return Case{}, false, err
	}
	if len(page.Cases) == 0 {
		return Case{}, false, nil
	}

	return page.Cases[0], true, nil
}

// FindOpenCaseByCompareOrigin returns the newest open compare-derived case for one exact compare lineage when it exists.
func (s *Service) FindOpenCaseByCompareOrigin(ctx context.Context, tenantID string, sourceEvalReportID string, compareOrigin CompareOrigin) (Case, bool, error) {
	return s.store.FindOpenByCompareOrigin(ctx, tenantID, sourceEvalReportID, compareOrigin)
}

// SummarizeBySourceEvalReportIDs returns follow-up case aggregates for source eval reports.
func (s *Service) SummarizeBySourceEvalReportIDs(ctx context.Context, tenantID string, reportIDs []string) (map[string]EvalReportFollowUpSummary, error) {
	return s.store.SummarizeBySourceEvalReportIDs(ctx, tenantID, reportIDs)
}

// SummarizeCompareOriginBySourceEvalReportIDs returns compare-derived follow-up case aggregates for source eval reports.
func (s *Service) SummarizeCompareOriginBySourceEvalReportIDs(ctx context.Context, tenantID string, reportIDs []string) (map[string]EvalReportCompareFollowUpSummary, error) {
	return s.store.SummarizeCompareOriginBySourceEvalReportIDs(ctx, tenantID, reportIDs)
}

// SummarizeBySourceEvalCaseIDs returns follow-up case aggregates for source eval cases.
func (s *Service) SummarizeBySourceEvalCaseIDs(ctx context.Context, tenantID string, evalCaseIDs []string) (map[string]EvalCaseFollowUpSummary, error) {
	return s.store.SummarizeBySourceEvalCaseIDs(ctx, tenantID, evalCaseIDs)
}

// ListCaseNotes returns recent append-only notes for a case.
func (s *Service) ListCaseNotes(ctx context.Context, caseID string, limit int) ([]Note, error) {
	return s.store.ListNotes(ctx, caseID, limit)
}

// CloseCase marks an operator case as closed.
func (s *Service) CloseCase(ctx context.Context, caseID string, closedBy string) (Case, error) {
	return s.store.Close(ctx, caseID, fallbackString(closedBy, "operator"), time.Now().UTC())
}

// ReopenCase returns a closed case back to the open queue.
func (s *Service) ReopenCase(ctx context.Context, caseID string, reopenedBy string) (Case, error) {
	reopenedAt := time.Now().UTC()
	return s.store.Reopen(ctx, caseID, fallbackString(strings.TrimSpace(reopenedBy), "operator"), reopenedAt)
}

// AssignCase assigns an open case to an operator using optimistic concurrency.
func (s *Service) AssignCase(ctx context.Context, existing Case, assignedTo string) (Case, error) {
	return s.store.Assign(
		ctx,
		existing.ID,
		fallbackString(strings.TrimSpace(assignedTo), "operator"),
		time.Now().UTC(),
		existing.UpdatedAt,
	)
}

// UnassignCase returns an assigned open case back to the shared queue using optimistic concurrency.
func (s *Service) UnassignCase(ctx context.Context, existing Case, unassignedBy string) (Case, error) {
	if existing.Status != StatusOpen || strings.TrimSpace(existing.AssignedTo) == "" {
		return Case{}, ErrInvalidCaseState
	}
	return s.store.Unassign(
		ctx,
		existing.ID,
		fallbackString(strings.TrimSpace(unassignedBy), "operator"),
		time.Now().UTC(),
		existing.UpdatedAt,
	)
}

// AddNote appends a durable operator note to an existing case.
func (s *Service) AddNote(ctx context.Context, existing Case, body string, createdBy string) (Note, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Note{}, ErrInvalidNote
	}

	now := time.Now().UTC()
	return s.store.AppendNote(ctx, Note{
		ID:        newCaseNoteID(now),
		TenantID:  existing.TenantID,
		CaseID:    existing.ID,
		Body:      body,
		CreatedBy: fallbackString(strings.TrimSpace(createdBy), "operator"),
		CreatedAt: now,
	})
}

func newCaseID(now time.Time) string {
	return fmt.Sprintf("case-%d-%d", now.UnixNano(), caseIDSequence.Add(1))
}

func newCaseNoteID(now time.Time) string {
	return fmt.Sprintf("case-note-%d-%d", now.UnixNano(), caseNoteIDSequence.Add(1))
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}
