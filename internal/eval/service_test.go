package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
	"opspilot-go/internal/observability/tracedetail"
)

func TestServicePromoteCaseBuildsLineageFromCaseAndTrace(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:             "case-1",
		TenantID:       "tenant-1",
		Title:          "Investigate workflow failure",
		Summary:        "Failure promoted for eval coverage.",
		SourceTaskID:   "task-1",
		SourceReportID: "report-1",
	}
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			if caseID != sourceCase.ID {
				return casesvc.Case{}, casesvc.ErrCaseNotFound
			}
			return sourceCase, nil
		}),
		traceLookupFunc(func(_ context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
			if input.CaseID != sourceCase.ID {
				return tracedetail.Result{}, tracedetail.ErrInvalidLookup
			}
			return tracedetail.Result{
				Lineage: tracedetail.Lineage{
					TaskID:   sourceCase.SourceTaskID,
					ReportID: sourceCase.SourceReportID,
				},
				TraceID:   "trace-1",
				VersionID: "version-1",
			}, nil
		}),
	)

	got, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
		OperatorNote: "promote this failure into regression coverage",
		CreatedBy:    "operator-1",
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	if !created {
		t.Fatal("created = false, want true")
	}
	if got.SourceTaskID != sourceCase.SourceTaskID {
		t.Fatalf("SourceTaskID = %q, want %q", got.SourceTaskID, sourceCase.SourceTaskID)
	}
	if got.SourceReportID != sourceCase.SourceReportID {
		t.Fatalf("SourceReportID = %q, want %q", got.SourceReportID, sourceCase.SourceReportID)
	}
	if got.TraceID != "trace-1" {
		t.Fatalf("TraceID = %q, want %q", got.TraceID, "trace-1")
	}
	if got.VersionID != "version-1" {
		t.Fatalf("VersionID = %q, want %q", got.VersionID, "version-1")
	}
}

func TestServicePromoteCaseIsIdempotentBySourceCase(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:       "case-2",
		TenantID: "tenant-1",
		Title:    "Case already promoted",
	}
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return sourceCase, nil
		}),
		traceLookupFunc(func(_ context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
			return tracedetail.Result{}, nil
		}),
	)

	first, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}
	if !created {
		t.Fatal("created(first) = false, want true")
	}

	second, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}
	if created {
		t.Fatal("created(second) = true, want false")
	}
	if second.ID != first.ID {
		t.Fatalf("second.ID = %q, want %q", second.ID, first.ID)
	}
}

func TestServicePromoteCaseRejectsTenantMismatch(t *testing.T) {
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return casesvc.Case{
				ID:       caseID,
				TenantID: "tenant-a",
				Title:    "Cross tenant",
			}, nil
		}),
		nil,
	)

	_, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-b",
		SourceCaseID: "case-cross-tenant",
	})
	if !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidSource)
	}
}

func TestServicePromoteCaseRejectsCrossTenantExistingRecord(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:       "case-cross-tenant-existing",
		TenantID: "tenant-a",
		Title:    "Cross tenant existing eval",
	}
	store := newMemoryStore()
	if _, err := store.Save(context.Background(), EvalCase{
		ID:           "eval-existing",
		TenantID:     "tenant-a",
		SourceCaseID: sourceCase.ID,
		Title:        sourceCase.Title,
		Summary:      "existing",
		CreatedBy:    "operator-a",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}
	service := NewServiceWithStore(store,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return sourceCase, nil
		}),
		nil,
	)

	_, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-b",
		SourceCaseID: sourceCase.ID,
	})
	if !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidSource)
	}
}

type caseReaderFunc func(ctx context.Context, caseID string) (casesvc.Case, error)

func (fn caseReaderFunc) GetCase(ctx context.Context, caseID string) (casesvc.Case, error) {
	return fn(ctx, caseID)
}

type traceLookupFunc func(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error)

func (fn traceLookupFunc) Lookup(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
	return fn(ctx, input)
}
