package version

import (
	"context"
	"errors"
	"time"
)

// Store persists durable runtime version snapshots.
type Store interface {
	Save(ctx context.Context, item Version) (Version, error)
	Get(ctx context.Context, versionID string) (Version, error)
	List(ctx context.Context, filter ListFilter) (ListPage, error)
}

// Service manages durable runtime version metadata.
type Service struct {
	store Store
}

// NewService constructs the version service with a memory-backed default store.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore constructs the version service with a caller-provided store.
func NewServiceWithStore(store Store) *Service {
	if store == nil {
		store = newMemoryStore()
	}

	return &Service{store: store}
}

// GetVersion returns a version by ID.
func (s *Service) GetVersion(ctx context.Context, versionID string) (Version, error) {
	if versionID == DefaultVersionID {
		return s.ensureCurrentVersion(ctx)
	}

	return s.store.Get(ctx, versionID)
}

// ListVersions returns a durable version page.
func (s *Service) ListVersions(ctx context.Context, filter ListFilter) (ListPage, error) {
	if _, err := s.ensureCurrentVersion(ctx); err != nil {
		return ListPage{}, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return s.store.List(ctx, filter)
}

// CurrentVersion ensures the built-in runtime version exists and returns it.
func (s *Service) CurrentVersion(ctx context.Context) (Version, error) {
	return s.ensureCurrentVersion(ctx)
}

// CurrentVersionID ensures the current version exists and returns its ID.
func (s *Service) CurrentVersionID(ctx context.Context) (string, error) {
	item, err := s.ensureCurrentVersion(ctx)
	if err != nil {
		return "", err
	}

	return item.ID, nil
}

func (s *Service) ensureCurrentVersion(ctx context.Context) (Version, error) {
	item, err := s.store.Get(ctx, DefaultVersionID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, ErrVersionNotFound) {
		return Version{}, err
	}

	return s.store.Save(ctx, defaultVersion())
}

func defaultVersion() Version {
	return Version{
		ID:                  DefaultVersionID,
		RuntimeVersion:      "runtime-skeleton-v1",
		Provider:            "",
		Model:               "",
		PromptBundle:        "prompt-skeleton-v1",
		PlannerVersion:      "planner-skeleton-v1",
		RetrievalVersion:    "retrieval-skeleton-v1",
		ToolRegistryVersion: "ticket-http-adapters-v1",
		CriticVersion:       "critic-skeleton-v1",
		WorkflowVersion:     "temporal-bridge-v1",
		Notes:               "Default runtime version for the current local skeleton.",
		CreatedAt:           time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC),
	}
}
