package cases

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type memoryStore struct {
	mu      sync.RWMutex
	records map[string]Case
	notes   map[string][]Note
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		records: map[string]Case{},
		notes:   map[string][]Note{},
	}
}

func (s *memoryStore) Save(_ context.Context, item Case) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[item.ID] = item
	return item, nil
}

func (s *memoryStore) Get(_ context.Context, caseID string) (Case, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}

	return item, nil
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) (ListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	items := make([]Case, 0, len(s.records))
	for _, item := range s.records {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.AssignedTo != "" && item.AssignedTo != filter.AssignedTo {
			continue
		}
		if filter.UnassignedOnly && item.AssignedTo != "" {
			continue
		}
		if filter.EvalBackedOnly && item.SourceEvalReportID == "" {
			continue
		}
		if filter.CompareOriginOnly && (item.CompareOrigin.LeftEvalReportID == "" || item.CompareOrigin.RightEvalReportID == "" || item.CompareOrigin.SelectedSide == "") {
			continue
		}
		if filter.ExcludeCompareOrigin && item.CompareOrigin.SelectedSide != "" {
			continue
		}
		if filter.SourceTaskID != "" && item.SourceTaskID != filter.SourceTaskID {
			continue
		}
		if filter.SourceReportID != "" && item.SourceReportID != filter.SourceReportID {
			continue
		}
		if filter.SourceEvalReportID != "" && item.SourceEvalReportID != filter.SourceEvalReportID {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID > items[j].ID
	})

	if offset >= len(items) {
		return ListPage{Cases: []Case{}}, nil
	}

	end := offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}

	page := ListPage{
		Cases:   append([]Case(nil), items[offset:end]...),
		HasMore: hasMore,
	}
	if hasMore {
		page.NextOffset = end
	}

	return page, nil
}

func (s *memoryStore) SummarizeBySourceEvalReportIDs(_ context.Context, tenantID string, reportIDs []string) (map[string]EvalReportFollowUpSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(reportIDs) == 0 {
		return map[string]EvalReportFollowUpSummary{}, nil
	}

	allowed := make(map[string]struct{}, len(reportIDs))
	for _, reportID := range reportIDs {
		if reportID == "" {
			continue
		}
		allowed[reportID] = struct{}{}
	}

	summaries := make(map[string]EvalReportFollowUpSummary, len(allowed))
	latestCases := make(map[string]Case, len(allowed))
	for _, item := range s.records {
		if item.TenantID != tenantID || item.SourceEvalReportID == "" {
			continue
		}
		if _, ok := allowed[item.SourceEvalReportID]; !ok {
			continue
		}

		summary := summaries[item.SourceEvalReportID]
		summary.SourceEvalReportID = item.SourceEvalReportID
		summary.FollowUpCaseCount++
		if item.Status == StatusOpen {
			summary.OpenFollowUpCaseCount++
		}
		latest, ok := latestCases[item.SourceEvalReportID]
		if !ok ||
			item.UpdatedAt.After(latest.UpdatedAt) ||
			(item.UpdatedAt.Equal(latest.UpdatedAt) && item.CreatedAt.After(latest.CreatedAt)) ||
			(item.UpdatedAt.Equal(latest.UpdatedAt) && item.CreatedAt.Equal(latest.CreatedAt) && item.ID > latest.ID) {
			latestCases[item.SourceEvalReportID] = item
			summary.LatestFollowUpCaseID = item.ID
			summary.LatestFollowUpCaseStatus = item.Status
		}
		summaries[item.SourceEvalReportID] = summary
	}

	for reportID := range allowed {
		if _, ok := summaries[reportID]; !ok {
			summaries[reportID] = EvalReportFollowUpSummary{SourceEvalReportID: reportID}
		}
	}

	return summaries, nil
}

func (s *memoryStore) AppendNote(_ context.Context, note Note) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[note.CaseID]
	if !ok {
		return Note{}, fmt.Errorf("%w: %s", ErrCaseNotFound, note.CaseID)
	}
	item.UpdatedAt = note.CreatedAt
	s.records[note.CaseID] = item
	s.notes[note.CaseID] = append(s.notes[note.CaseID], note)

	return note, nil
}

func (s *memoryStore) ListNotes(_ context.Context, caseID string, limit int) ([]Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.records[caseID]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if limit <= 0 {
		limit = 20
	}

	source := s.notes[caseID]
	if len(source) == 0 {
		return []Note{}, nil
	}

	count := len(source)
	if count > limit {
		count = limit
	}

	items := make([]Note, 0, count)
	for i := len(source) - 1; i >= 0 && len(items) < count; i-- {
		items = append(items, source[i])
	}

	return items, nil
}

func (s *memoryStore) Close(_ context.Context, caseID string, closedBy string, closedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status == StatusClosed {
		return Case{}, ErrInvalidCaseState
	}

	item.Status = StatusClosed
	item.ClosedBy = closedBy
	item.UpdatedAt = closedAt
	s.records[caseID] = item

	return item, nil
}

func (s *memoryStore) Reopen(_ context.Context, caseID string, reopenedBy string, reopenedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status != StatusClosed {
		return Case{}, ErrInvalidCaseState
	}

	item.Status = StatusOpen
	item.ClosedBy = ""
	item.UpdatedAt = reopenedAt
	s.records[caseID] = item
	s.notes[caseID] = append(s.notes[caseID], Note{
		ID:        newCaseNoteID(reopenedAt),
		TenantID:  item.TenantID,
		CaseID:    item.ID,
		Body:      fmt.Sprintf("case reopened by %s", reopenedBy),
		CreatedBy: reopenedBy,
		CreatedAt: reopenedAt,
	})

	return item, nil
}

func (s *memoryStore) Assign(_ context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status == StatusClosed {
		return Case{}, ErrInvalidCaseState
	}
	if !item.UpdatedAt.Equal(expectedUpdatedAt) {
		return Case{}, ErrCaseConflict
	}

	item.AssignedTo = assignedTo
	item.AssignedAt = assignedAt
	item.UpdatedAt = assignedAt
	s.records[caseID] = item

	return item, nil
}

func (s *memoryStore) Unassign(_ context.Context, caseID string, unassignedBy string, unassignedAt time.Time, expectedUpdatedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status == StatusClosed {
		return Case{}, ErrInvalidCaseState
	}
	if item.AssignedTo == "" {
		return Case{}, ErrInvalidCaseState
	}
	if !item.UpdatedAt.Equal(expectedUpdatedAt) {
		return Case{}, ErrCaseConflict
	}

	item.AssignedTo = ""
	item.AssignedAt = time.Time{}
	item.UpdatedAt = unassignedAt
	s.records[caseID] = item
	s.notes[caseID] = append(s.notes[caseID], Note{
		ID:        newCaseNoteID(unassignedAt),
		TenantID:  item.TenantID,
		CaseID:    item.ID,
		Body:      fmt.Sprintf("case returned to queue by %s", unassignedBy),
		CreatedBy: unassignedBy,
		CreatedAt: unassignedAt,
	})

	return item, nil
}
