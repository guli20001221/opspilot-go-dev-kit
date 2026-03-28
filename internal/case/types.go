package cases

import (
	"errors"
	"time"
)

const (
	// StatusOpen identifies a case that still needs operator follow-up.
	StatusOpen = "open"
	// StatusClosed identifies a case that no longer needs operator follow-up.
	StatusClosed = "closed"
)

// ErrCaseNotFound identifies missing case records.
var ErrCaseNotFound = errors.New("case not found")

// ErrInvalidCaseState identifies invalid case state transitions.
var ErrInvalidCaseState = errors.New("invalid case state")

// ErrCaseConflict identifies stale writes against an updated case row.
var ErrCaseConflict = errors.New("case conflict")

// ErrInvalidNote identifies invalid case note payloads.
var ErrInvalidNote = errors.New("invalid case note")

// Case is the durable read model for an operator-managed case.
type Case struct {
	ID                 string
	TenantID           string
	Status             string
	Title              string
	Summary            string
	SourceTaskID       string
	SourceReportID     string
	SourceEvalReportID string
	SourceEvalCaseID   string
	CompareOrigin      CompareOrigin
	CreatedBy          string
	AssignedTo         string
	AssignedAt         time.Time
	ClosedBy           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CompareOrigin captures structured lineage for cases created from an eval-report comparison.
type CompareOrigin struct {
	LeftEvalReportID  string
	RightEvalReportID string
	SelectedSide      string
}

// Note is an append-only operator note attached to a case.
type Note struct {
	ID        string
	TenantID  string
	CaseID    string
	Body      string
	CreatedBy string
	CreatedAt time.Time
}

// CreateInput is the typed case creation request.
type CreateInput struct {
	TenantID           string
	Title              string
	Summary            string
	SourceTaskID       string
	SourceReportID     string
	SourceEvalReportID string
	SourceEvalCaseID   string
	CompareOrigin      CompareOrigin
	CreatedBy          string
}

// ListFilter narrows case list queries for operator-facing views.
type ListFilter struct {
	TenantID             string
	Status               string
	AssignedTo           string
	UnassignedOnly       bool
	EvalBackedOnly       bool
	CompareOriginOnly    bool
	ExcludeCompareOrigin bool
	SourceTaskID         string
	SourceReportID       string
	SourceEvalReportID   string
	SourceEvalCaseID     string
	Limit                int
	Offset               int
}

// ListPage is the paginated case list result.
type ListPage struct {
	Cases      []Case
	HasMore    bool
	NextOffset int
}

// EvalReportFollowUpSummary aggregates case follow-up state for one source eval report.
type EvalReportFollowUpSummary struct {
	SourceEvalReportID       string
	FollowUpCaseCount        int
	OpenFollowUpCaseCount    int
	LatestFollowUpCaseID     string
	LatestFollowUpCaseStatus string
}

// EvalCaseFollowUpSummary aggregates case follow-up state for one source eval case.
type EvalCaseFollowUpSummary struct {
	SourceEvalCaseID         string
	FollowUpCaseCount        int
	OpenFollowUpCaseCount    int
	LatestFollowUpCaseID     string
	LatestFollowUpCaseStatus string
}
