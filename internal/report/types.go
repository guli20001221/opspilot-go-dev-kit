package report

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	// TypeWorkflowSummary identifies reports derived from workflow task output.
	TypeWorkflowSummary = "workflow_summary"

	// StatusReady identifies a report ready for operator consumption.
	StatusReady = "ready"
)

// ErrReportNotFound identifies missing report records.
var ErrReportNotFound = errors.New("report not found")

// Report is the durable read model for a generated report.
type Report struct {
	ID           string
	TenantID     string
	SourceTaskID string
	VersionID    string
	ReportType   string
	Status       string
	Title        string
	Summary      string
	ContentURI   string
	MetadataJSON json.RawMessage
	CreatedBy    string
	CreatedAt    time.Time
	ReadyAt      *time.Time
}

// ListFilter constrains report list reads.
type ListFilter struct {
	TenantID     string
	Status       string
	ReportType   string
	SourceTaskID string
	Limit        int
	Offset       int
}

// ListPage is a single report list page.
type ListPage struct {
	Reports    []Report
	HasMore    bool
	NextOffset int
}

// ComparisonSummary captures the operator-facing differences between two reports.
type ComparisonSummary struct {
	SameTenant         bool
	SameReportType     bool
	VersionChanged     bool
	SourceTaskChanged  bool
	TitleChanged       bool
	SummaryChanged     bool
	ContentURIChanged  bool
	MetadataChanged    bool
	CreatedAtChanged   bool
	ReadyAtChanged     bool
	ReadyAtDeltaSecond int64
}

// Comparison holds two durable reports and their derived comparison summary.
type Comparison struct {
	Left    Report
	Right   Report
	Summary ComparisonSummary
}
