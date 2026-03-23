package cases

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

var caseIDSequence atomic.Uint64

// Store persists case read models.
type Store interface {
	Save(ctx context.Context, item Case) (Case, error)
	Get(ctx context.Context, caseID string) (Case, error)
	List(ctx context.Context, filter ListFilter) (ListPage, error)
	Close(ctx context.Context, caseID string, closedBy string, closedAt time.Time) (Case, error)
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
		ID:             newCaseID(now),
		TenantID:       input.TenantID,
		Status:         StatusOpen,
		Title:          input.Title,
		Summary:        input.Summary,
		SourceTaskID:   input.SourceTaskID,
		SourceReportID: input.SourceReportID,
		CreatedBy:      fallbackString(input.CreatedBy, "operator"),
		CreatedAt:      now,
		UpdatedAt:      now,
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

// CloseCase marks an operator case as closed.
func (s *Service) CloseCase(ctx context.Context, caseID string, closedBy string) (Case, error) {
	return s.store.Close(ctx, caseID, fallbackString(closedBy, "operator"), time.Now().UTC())
}

func newCaseID(now time.Time) string {
	return fmt.Sprintf("case-%d-%d", now.UnixNano(), caseIDSequence.Add(1))
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}
