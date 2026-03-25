package eval

import (
	"errors"
	"time"
)

// ErrEvalCaseNotFound identifies missing durable eval case records.
var ErrEvalCaseNotFound = errors.New("eval case not found")

// ErrInvalidSource identifies invalid source lineage for eval promotion.
var ErrInvalidSource = errors.New("invalid eval source")

// ErrEvalCaseExists identifies duplicate promotion of the same source case.
var ErrEvalCaseExists = errors.New("eval case already exists")

// EvalCase is the durable read model for a promoted evaluation case.
type EvalCase struct {
	ID             string
	TenantID       string
	SourceCaseID   string
	SourceTaskID   string
	SourceReportID string
	TraceID        string
	VersionID      string
	Title          string
	Summary        string
	OperatorNote   string
	CreatedBy      string
	CreatedAt      time.Time
}

// ListFilter constrains eval-case list reads.
type ListFilter struct {
	TenantID       string
	SourceCaseID   string
	SourceTaskID   string
	SourceReportID string
	VersionID      string
	Limit          int
	Offset         int
}

// ListPage is a single eval-case list page.
type ListPage struct {
	EvalCases  []EvalCase
	HasMore    bool
	NextOffset int
}

// CreateInput is the typed eval case promotion request.
type CreateInput struct {
	TenantID     string
	SourceCaseID string
	OperatorNote string
	CreatedBy    string
}
