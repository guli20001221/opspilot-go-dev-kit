package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	casesvc "opspilot-go/internal/case"
	"opspilot-go/internal/observability/tracedetail"
)

var evalCaseIDSequence atomic.Uint64

// Store persists durable eval case records.
type Store interface {
	Save(ctx context.Context, item EvalCase) (EvalCase, error)
	Get(ctx context.Context, evalCaseID string) (EvalCase, error)
	GetBySourceCase(ctx context.Context, sourceCaseID string) (EvalCase, error)
	List(ctx context.Context, filter ListFilter) (ListPage, error)
}

type caseReader interface {
	GetCase(ctx context.Context, caseID string) (casesvc.Case, error)
}

type evalCaseFollowUpSummarizer interface {
	SummarizeBySourceEvalCaseIDs(ctx context.Context, tenantID string, evalCaseIDs []string) (map[string]casesvc.EvalCaseFollowUpSummary, error)
}

type traceLookup interface {
	Lookup(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error)
}

// Service manages durable eval case promotion from operator cases.
type Service struct {
	store     Store
	cases     caseReader
	summaries evalCaseFollowUpSummarizer
	traces    traceLookup
}

// NewService constructs the eval service with in-memory defaults.
func NewService(cases caseReader, traces traceLookup) *Service {
	return NewServiceWithStore(nil, cases, traces)
}

// NewServiceWithStore constructs the eval service with caller-provided storage.
func NewServiceWithStore(store Store, cases caseReader, traces traceLookup) *Service {
	if store == nil {
		store = newMemoryStore()
	}

	var summaries evalCaseFollowUpSummarizer
	if candidate, ok := cases.(evalCaseFollowUpSummarizer); ok {
		summaries = candidate
	}

	return &Service{
		store:     store,
		cases:     cases,
		summaries: summaries,
		traces:    traces,
	}
}

// PromoteCase creates or reuses a durable eval case from an operator case.
func (s *Service) PromoteCase(ctx context.Context, input CreateInput) (EvalCase, bool, error) {
	sourceCase, err := s.cases.GetCase(ctx, input.SourceCaseID)
	if err != nil {
		return EvalCase{}, false, err
	}
	if sourceCase.TenantID != input.TenantID {
		return EvalCase{}, false, ErrInvalidSource
	}
	if existing, err := s.store.GetBySourceCase(ctx, input.SourceCaseID); err == nil {
		enriched, enrichErr := s.enrichEvalCase(ctx, existing)
		if enrichErr != nil {
			return EvalCase{}, false, enrichErr
		}
		return enriched, false, nil
	} else if !errors.Is(err, ErrEvalCaseNotFound) {
		return EvalCase{}, false, err
	}

	item := EvalCase{
		ID:             newEvalCaseID(time.Now().UTC()),
		TenantID:       input.TenantID,
		SourceCaseID:   sourceCase.ID,
		SourceTaskID:   sourceCase.SourceTaskID,
		SourceReportID: sourceCase.SourceReportID,
		Title:          sourceCase.Title,
		Summary:        sourceCase.Summary,
		OperatorNote:   strings.TrimSpace(input.OperatorNote),
		CreatedBy:      fallbackString(strings.TrimSpace(input.CreatedBy), "operator"),
		CreatedAt:      time.Now().UTC(),
	}
	if item.Summary == "" {
		item.Summary = fmt.Sprintf("Promoted from case %s", sourceCase.ID)
	}
	if s.traces != nil {
		traceResult, err := s.traces.Lookup(ctx, tracedetail.LookupInput{CaseID: sourceCase.ID})
		if err == nil {
			if item.SourceTaskID == "" {
				item.SourceTaskID = traceResult.Lineage.TaskID
			}
			if item.SourceReportID == "" {
				item.SourceReportID = traceResult.Lineage.ReportID
			}
			item.TraceID = traceResult.TraceID
			item.VersionID = traceResult.VersionID
		}
	}

	saved, err := s.store.Save(ctx, item)
	if err != nil {
		if errors.Is(err, ErrEvalCaseExists) {
			existing, getErr := s.store.GetBySourceCase(ctx, input.SourceCaseID)
			if getErr != nil {
				return EvalCase{}, false, getErr
			}
			enriched, enrichErr := s.enrichEvalCase(ctx, existing)
			if enrichErr != nil {
				return EvalCase{}, false, enrichErr
			}
			return enriched, false, nil
		}
		return EvalCase{}, false, err
	}

	enriched, err := s.enrichEvalCase(ctx, saved)
	if err != nil {
		return EvalCase{}, false, err
	}
	return enriched, true, nil
}

// GetEvalCase returns a durable eval case by ID.
func (s *Service) GetEvalCase(ctx context.Context, evalCaseID string) (EvalCase, error) {
	item, err := s.store.Get(ctx, evalCaseID)
	if err != nil {
		return EvalCase{}, err
	}
	return s.enrichEvalCase(ctx, item)
}

// ListEvalCases returns one durable eval-case page.
func (s *Service) ListEvalCases(ctx context.Context, filter ListFilter) (ListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.NeedsFollowUp != nil {
		return s.listEvalCasesWithFollowUpFilter(ctx, filter)
	}

	page, err := s.store.List(ctx, filter)
	if err != nil {
		return ListPage{}, err
	}
	page.EvalCases, err = s.applyFollowUpSummaries(ctx, page.EvalCases)
	if err != nil {
		return ListPage{}, err
	}
	return page, nil
}

func (s *Service) listEvalCasesWithFollowUpFilter(ctx context.Context, filter ListFilter) (ListPage, error) {
	batchSize := filter.Limit
	if batchSize < 20 {
		batchSize = 20
	}

	baseFilter := filter
	baseFilter.NeedsFollowUp = nil
	baseFilter.Offset = 0
	baseFilter.Limit = batchSize

	page := ListPage{EvalCases: make([]EvalCase, 0, filter.Limit)}
	matchedCount := 0
	rawOffset := 0

	for {
		baseFilter.Offset = rawOffset
		rawPage, err := s.store.List(ctx, baseFilter)
		if err != nil {
			return ListPage{}, err
		}
		if len(rawPage.EvalCases) == 0 {
			return page, nil
		}

		enriched, err := s.applyFollowUpSummaries(ctx, rawPage.EvalCases)
		if err != nil {
			return ListPage{}, err
		}

		for _, item := range enriched {
			hasOpenFollowUp := item.OpenFollowUpCaseCount > 0
			if hasOpenFollowUp != *filter.NeedsFollowUp {
				continue
			}
			if matchedCount < filter.Offset {
				matchedCount++
				continue
			}
			if len(page.EvalCases) < filter.Limit {
				page.EvalCases = append(page.EvalCases, item)
				matchedCount++
				continue
			}

			page.HasMore = true
			page.NextOffset = filter.Offset + len(page.EvalCases)
			return page, nil
		}

		if !rawPage.HasMore {
			return page, nil
		}
		rawOffset = rawPage.NextOffset
	}
}

func (s *Service) applyFollowUpSummaries(ctx context.Context, items []EvalCase) ([]EvalCase, error) {
	if len(items) == 0 || s.summaries == nil {
		return items, nil
	}

	tenantID := items[0].TenantID
	evalCaseIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.ID == "" || item.TenantID != tenantID {
			continue
		}
		evalCaseIDs = append(evalCaseIDs, item.ID)
	}
	if len(evalCaseIDs) == 0 {
		return items, nil
	}

	summaries, err := s.summaries.SummarizeBySourceEvalCaseIDs(ctx, tenantID, evalCaseIDs)
	if err != nil {
		return nil, err
	}
	enriched := append([]EvalCase(nil), items...)
	for i := range enriched {
		summary := summaries[enriched[i].ID]
		enriched[i].FollowUpCaseCount = summary.FollowUpCaseCount
		enriched[i].OpenFollowUpCaseCount = summary.OpenFollowUpCaseCount
		enriched[i].LatestFollowUpCaseID = summary.LatestFollowUpCaseID
		enriched[i].LatestFollowUpCaseStatus = summary.LatestFollowUpCaseStatus
	}
	return enriched, nil
}

func (s *Service) enrichEvalCase(ctx context.Context, item EvalCase) (EvalCase, error) {
	items, err := s.applyFollowUpSummaries(ctx, []EvalCase{item})
	if err != nil {
		return EvalCase{}, err
	}
	return items[0], nil
}

func newEvalCaseID(now time.Time) string {
	return fmt.Sprintf("eval-case-%d-%d", now.UnixNano(), evalCaseIDSequence.Add(1))
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}
