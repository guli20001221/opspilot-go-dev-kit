package taskboard

import "time"

// TaskBoard is the admin-facing read model for the current task slice.
type TaskBoard struct {
	Items   []TaskItem
	Page    PageInfo
	Summary Summary
}

// TaskItem is the operator-facing task row used by admin pages.
type TaskItem struct {
	TaskID           string
	RequestID        string
	TenantID         string
	SessionID        string
	TaskType         string
	Status           string
	Reason           string
	ErrorReason      string
	AuditRef         string
	RequiresApproval bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PageInfo carries list pagination metadata for the admin view.
type PageInfo struct {
	HasMore    bool
	NextOffset *int
}

// Summary aggregates the currently visible task slice.
type Summary struct {
	VisibleCount          int
	RequiresApprovalCount int
	StatusCounts          StatusCounts
	ReasonCounts          ReasonCounts
	TaskTypeCounts        TaskTypeCounts
	LatestUpdatedAt       *time.Time
	LatestFailureReason   string
}

// StatusCounts aggregates visible task statuses.
type StatusCounts struct {
	Queued          int
	Running         int
	Succeeded       int
	Failed          int
	WaitingApproval int
}

// ReasonCounts aggregates visible task promotion reasons.
type ReasonCounts struct {
	WorkflowRequired int
	ApprovalRequired int
}

// TaskTypeCounts aggregates visible task types.
type TaskTypeCounts struct {
	ReportGeneration      int
	ApprovedToolExecution int
}
