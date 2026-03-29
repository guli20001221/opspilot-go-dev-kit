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

func (s *memoryStore) SaveOrReuseOpenEvalRunCase(_ context.Context, item Case) (Case, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.records {
		if existing.TenantID != item.TenantID || existing.Status != StatusOpen || existing.SourceEvalRunID != item.SourceEvalRunID {
			continue
		}
		if caseSortsAfter(existing, item) {
			return existing, false, nil
		}
		return existing, false, nil
	}

	s.records[item.ID] = item
	return item, true, nil
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
	allowedEvalReports := make(map[string]struct{}, len(filter.SourceEvalReportIDs))
	for _, reportID := range filter.SourceEvalReportIDs {
		if reportID == "" {
			continue
		}
		allowedEvalReports[reportID] = struct{}{}
	}
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
		if filter.RunBackedOnly && item.SourceEvalRunID == "" {
			continue
		}
		if filter.CompareOriginOnly && (item.CompareOrigin.LeftEvalReportID == "" || item.CompareOrigin.RightEvalReportID == "" || item.CompareOrigin.SelectedSide == "") {
			continue
		}
		if filter.ExcludeCompareOrigin && item.CompareOrigin.SelectedSide != "" {
			continue
		}
		if filter.PlainEvalReportOnly && item.SourceEvalCaseID != "" {
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
		if len(allowedEvalReports) > 0 {
			if _, ok := allowedEvalReports[item.SourceEvalReportID]; !ok {
				continue
			}
		}
		if filter.SourceEvalCaseID != "" && item.SourceEvalCaseID != filter.SourceEvalCaseID {
			continue
		}
		if filter.SourceEvalRunID != "" && item.SourceEvalRunID != filter.SourceEvalRunID {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return caseSortsAfter(items[i], items[j])
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

func (s *memoryStore) FindOpenByCompareOrigin(_ context.Context, tenantID string, sourceEvalReportID string, compareOrigin CompareOrigin) (Case, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var latest Case
	found := false
	for _, item := range s.records {
		if item.TenantID != tenantID || item.Status != StatusOpen || item.SourceEvalReportID != sourceEvalReportID {
			continue
		}
		if item.CompareOrigin.LeftEvalReportID != compareOrigin.LeftEvalReportID ||
			item.CompareOrigin.RightEvalReportID != compareOrigin.RightEvalReportID ||
			item.CompareOrigin.SelectedSide != compareOrigin.SelectedSide {
			continue
		}
		if !found || caseSortsAfter(item, latest) {
			latest = item
			found = true
		}
	}
	if !found {
		return Case{}, false, nil
	}

	return latest, true, nil
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
		if !ok || caseSortsAfter(item, latest) {
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

func (s *memoryStore) SummarizeCompareOriginBySourceEvalReportIDs(_ context.Context, tenantID string, reportIDs []string) (map[string]EvalReportCompareFollowUpSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(reportIDs) == 0 {
		return map[string]EvalReportCompareFollowUpSummary{}, nil
	}

	allowed := make(map[string]struct{}, len(reportIDs))
	for _, reportID := range reportIDs {
		if reportID == "" {
			continue
		}
		allowed[reportID] = struct{}{}
	}

	summaries := make(map[string]EvalReportCompareFollowUpSummary, len(allowed))
	latestCases := make(map[string]Case, len(allowed))
	for _, item := range s.records {
		if item.TenantID != tenantID || item.SourceEvalReportID == "" {
			continue
		}
		if item.CompareOrigin.LeftEvalReportID == "" || item.CompareOrigin.RightEvalReportID == "" || item.CompareOrigin.SelectedSide == "" {
			continue
		}
		if _, ok := allowed[item.SourceEvalReportID]; !ok {
			continue
		}

		summary := summaries[item.SourceEvalReportID]
		summary.SourceEvalReportID = item.SourceEvalReportID
		summary.CompareFollowUpCaseCount++
		if item.Status == StatusOpen {
			summary.OpenCompareFollowUpCaseCount++
		}
		latest, ok := latestCases[item.SourceEvalReportID]
		if !ok || caseSortsAfter(item, latest) {
			latestCases[item.SourceEvalReportID] = item
			summary.LatestCompareFollowUpCaseID = item.ID
			summary.LatestCompareFollowUpCaseStatus = item.Status
		}
		summaries[item.SourceEvalReportID] = summary
	}

	for reportID := range allowed {
		if _, ok := summaries[reportID]; !ok {
			summaries[reportID] = EvalReportCompareFollowUpSummary{SourceEvalReportID: reportID}
		}
	}

	return summaries, nil
}

func (s *memoryStore) SummarizeBySourceEvalCaseIDs(_ context.Context, tenantID string, evalCaseIDs []string) (map[string]EvalCaseFollowUpSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(evalCaseIDs) == 0 {
		return map[string]EvalCaseFollowUpSummary{}, nil
	}

	allowed := make(map[string]struct{}, len(evalCaseIDs))
	for _, evalCaseID := range evalCaseIDs {
		if evalCaseID == "" {
			continue
		}
		allowed[evalCaseID] = struct{}{}
	}

	summaries := make(map[string]EvalCaseFollowUpSummary, len(allowed))
	latestCases := make(map[string]Case, len(allowed))
	for _, item := range s.records {
		if item.TenantID != tenantID || item.SourceEvalCaseID == "" {
			continue
		}
		if _, ok := allowed[item.SourceEvalCaseID]; !ok {
			continue
		}

		summary := summaries[item.SourceEvalCaseID]
		summary.SourceEvalCaseID = item.SourceEvalCaseID
		summary.FollowUpCaseCount++
		if item.Status == StatusOpen {
			summary.OpenFollowUpCaseCount++
		}
		latest, ok := latestCases[item.SourceEvalCaseID]
		if !ok || caseSortsAfter(item, latest) {
			latestCases[item.SourceEvalCaseID] = item
			summary.LatestFollowUpCaseID = item.ID
			summary.LatestFollowUpCaseStatus = item.Status
		}
		summaries[item.SourceEvalCaseID] = summary
	}

	for evalCaseID := range allowed {
		if _, ok := summaries[evalCaseID]; !ok {
			summaries[evalCaseID] = EvalCaseFollowUpSummary{SourceEvalCaseID: evalCaseID}
		}
	}

	return summaries, nil
}

func (s *memoryStore) SummarizeBySourceEvalRunIDs(_ context.Context, tenantID string, runIDs []string) (map[string]EvalRunFollowUpSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(runIDs) == 0 {
		return map[string]EvalRunFollowUpSummary{}, nil
	}

	allowed := make(map[string]struct{}, len(runIDs))
	for _, runID := range runIDs {
		if runID == "" {
			continue
		}
		allowed[runID] = struct{}{}
	}

	summaries := make(map[string]EvalRunFollowUpSummary, len(allowed))
	latestCases := make(map[string]Case, len(allowed))
	for _, item := range s.records {
		if item.TenantID != tenantID || item.SourceEvalRunID == "" {
			continue
		}
		if _, ok := allowed[item.SourceEvalRunID]; !ok {
			continue
		}

		summary := summaries[item.SourceEvalRunID]
		summary.SourceEvalRunID = item.SourceEvalRunID
		summary.FollowUpCaseCount++
		if item.Status == StatusOpen {
			summary.OpenFollowUpCaseCount++
		}
		latest, ok := latestCases[item.SourceEvalRunID]
		if !ok || caseSortsAfter(item, latest) {
			latestCases[item.SourceEvalRunID] = item
			summary.LatestFollowUpCaseID = item.ID
			summary.LatestFollowUpCaseStatus = item.Status
			summary.LatestFollowUpAssignedTo = item.AssignedTo
		}
		summaries[item.SourceEvalRunID] = summary
	}

	for runID := range allowed {
		if _, ok := summaries[runID]; !ok {
			summaries[runID] = EvalRunFollowUpSummary{SourceEvalRunID: runID}
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
