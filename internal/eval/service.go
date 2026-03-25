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

type traceLookup interface {
	Lookup(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error)
}

// Service manages durable eval case promotion from operator cases.
type Service struct {
	store  Store
	cases  caseReader
	traces traceLookup
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

	return &Service{
		store:  store,
		cases:  cases,
		traces: traces,
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
		return existing, false, nil
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
			return existing, false, nil
		}
		return EvalCase{}, false, err
	}

	return saved, true, nil
}

// GetEvalCase returns a durable eval case by ID.
func (s *Service) GetEvalCase(ctx context.Context, evalCaseID string) (EvalCase, error) {
	return s.store.Get(ctx, evalCaseID)
}

// ListEvalCases returns one durable eval-case page.
func (s *Service) ListEvalCases(ctx context.Context, filter ListFilter) (ListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return s.store.List(ctx, filter)
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
