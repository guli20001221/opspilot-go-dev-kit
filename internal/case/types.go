package cases

import (
	"errors"
	"time"
)

const (
	// StatusOpen identifies a case that still needs operator follow-up.
	StatusOpen = "open"
)

// ErrCaseNotFound identifies missing case records.
var ErrCaseNotFound = errors.New("case not found")

// Case is the durable read model for an operator-managed case.
type Case struct {
	ID             string
	TenantID       string
	Status         string
	Title          string
	Summary        string
	SourceTaskID   string
	SourceReportID string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateInput is the typed case creation request.
type CreateInput struct {
	TenantID       string
	Title          string
	Summary        string
	SourceTaskID   string
	SourceReportID string
	CreatedBy      string
}
