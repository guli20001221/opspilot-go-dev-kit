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
