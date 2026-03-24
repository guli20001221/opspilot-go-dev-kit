package tracedetail

const (
	// SubjectTask identifies a workflow task trace subject.
	SubjectTask = "task"
	// SubjectReport identifies a durable report trace subject.
	SubjectReport = "report"
	// SubjectCase identifies a durable case trace subject.
	SubjectCase = "case"
)

// LookupInput identifies which durable object should be expanded into trace context.
type LookupInput struct {
	TaskID   string
	ReportID string
	CaseID   string
}

// Subject is the resolved durable object that initiated the drill-down request.
type Subject struct {
	Kind     string
	ID       string
	TenantID string
}

// Lineage captures durable links between task, report, and case records.
type Lineage struct {
	TaskID   string
	ReportID string
	CaseID   string
}

// TemporalRef is the Temporal workflow/run encoded in current audit provenance.
type TemporalRef struct {
	WorkflowID string
	RunID      string
}

// Result is the read-only trace drill-down view consumed by operator pages.
type Result struct {
	Subject      Subject
	Lineage      Lineage
	RequestID    string
	SessionID    string
	TraceID      string
	AuditRef     string
	TaskStatus   string
	ReportType   string
	ReportStatus string
	CaseStatus   string
	Temporal     *TemporalRef
	Warnings     []string
}
