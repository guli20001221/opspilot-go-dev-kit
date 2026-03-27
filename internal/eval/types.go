package eval

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrEvalCaseNotFound identifies missing durable eval case records.
var ErrEvalCaseNotFound = errors.New("eval case not found")

// ErrInvalidSource identifies invalid source lineage for eval promotion.
var ErrInvalidSource = errors.New("invalid eval source")

// ErrEvalCaseExists identifies duplicate promotion of the same source case.
var ErrEvalCaseExists = errors.New("eval case already exists")

// ErrEvalDatasetNotFound identifies missing durable eval dataset records.
var ErrEvalDatasetNotFound = errors.New("eval dataset not found")

// ErrInvalidEvalDataset identifies invalid eval dataset requests.
var ErrInvalidEvalDataset = errors.New("invalid eval dataset")

// ErrInvalidEvalDatasetState identifies invalid dataset lifecycle transitions.
var ErrInvalidEvalDatasetState = errors.New("invalid eval dataset state")

// ErrEvalRunNotFound identifies missing durable eval run records.
var ErrEvalRunNotFound = errors.New("eval run not found")

// ErrInvalidEvalRunState identifies invalid eval-run lifecycle transitions.
var ErrInvalidEvalRunState = errors.New("invalid eval run state")

// ErrEvalReportNotFound identifies missing durable eval-report records.
var ErrEvalReportNotFound = errors.New("eval report not found")

const (
	// DatasetStatusDraft identifies a draft dataset that is not yet active in regression runs.
	DatasetStatusDraft = "draft"
	// DatasetStatusPublished identifies an immutable dataset baseline ready for regression use.
	DatasetStatusPublished = "published"
)

const (
	// RunStatusQueued identifies an eval run that has been created but not yet executed.
	RunStatusQueued = "queued"
	// RunStatusRunning identifies an eval run currently executing.
	RunStatusRunning = "running"
	// RunStatusSucceeded identifies a completed eval run.
	RunStatusSucceeded = "succeeded"
	// RunStatusFailed identifies a failed eval run.
	RunStatusFailed = "failed"
)

const (
	// RunEventCreated identifies eval-run creation.
	RunEventCreated = "created"
	// RunEventClaimed identifies worker claim of a queued eval run.
	RunEventClaimed = "claimed"
	// RunEventSucceeded identifies successful eval-run completion.
	RunEventSucceeded = "succeeded"
	// RunEventFailed identifies failed eval-run completion.
	RunEventFailed = "failed"
	// RunEventRetried identifies operator re-queue of a failed eval run.
	RunEventRetried = "retried"
)

const (
	// RunItemResultSucceeded identifies a placeholder successful per-item outcome.
	RunItemResultSucceeded = "succeeded"
	// RunItemResultFailed identifies a placeholder failed per-item outcome.
	RunItemResultFailed = "failed"
)

const (
	// RunItemVerdictPass identifies a passing judge verdict.
	RunItemVerdictPass = "pass"
	// RunItemVerdictFail identifies a failing judge verdict.
	RunItemVerdictFail = "fail"
	// PlaceholderJudgeKind identifies the built-in deterministic judge implementation.
	PlaceholderJudgeKind = "placeholder"
	// PlaceholderJudgeVersion identifies the current built-in deterministic judge version.
	PlaceholderJudgeVersion = "placeholder-v1"
)

const (
	// EvalReportStatusReady identifies a materialized eval report ready for operator consumption.
	EvalReportStatusReady = "ready"
)

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

// EvalDatasetItem is one durable dataset membership row backed by a promoted eval case.
type EvalDatasetItem struct {
	EvalCaseID     string
	Title          string
	SourceCaseID   string
	SourceTaskID   string
	SourceReportID string
	TraceID        string
	VersionID      string
}

// EvalDataset is the durable read model for a draft or active evaluation dataset.
type EvalDataset struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	Status      string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedBy string
	PublishedAt time.Time
	Items       []EvalDatasetItem
}

// EvalDatasetSummary is the lightweight list projection for one durable eval dataset.
type EvalDatasetSummary struct {
	ID        string
	TenantID  string
	Name      string
	Status    string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	ItemCount int
}

// DatasetListFilter constrains eval-dataset list reads.
type DatasetListFilter struct {
	TenantID  string
	Status    string
	CreatedBy string
	Limit     int
	Offset    int
}

// DatasetListPage is one eval-dataset list page.
type DatasetListPage struct {
	Datasets   []EvalDatasetSummary
	HasMore    bool
	NextOffset int
}

// EvalRun is the durable read model for one eval-run kickoff.
type EvalRun struct {
	ID               string
	TenantID         string
	DatasetID        string
	DatasetName      string
	DatasetItemCount int
	ResultSummary    *EvalRunResultSummary
	Status           string
	CreatedBy        string
	ErrorReason      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        time.Time
	FinishedAt       time.Time
}

// EvalRunEvent is one append-only lifecycle event for a durable eval run.
type EvalRunEvent struct {
	ID        int64
	RunID     string
	Action    string
	Actor     string
	Detail    string
	CreatedAt time.Time
}

// EvalRunItem is one immutable membership row snapped onto a durable eval run.
type EvalRunItem struct {
	EvalCaseID     string
	Title          string
	SourceCaseID   string
	SourceTaskID   string
	SourceReportID string
	TraceID        string
	VersionID      string
}

// EvalRunDetail is the canonical detail read for one durable eval run.
type EvalRunDetail struct {
	Run         EvalRun
	Events      []EvalRunEvent
	Items       []EvalRunItem
	ItemResults []EvalRunItemResult
}

// EvalRunResultSummary captures operator-facing placeholder result counts for one run.
type EvalRunResultSummary struct {
	TotalItems      int
	RecordedResults int
	SucceededItems  int
	FailedItems     int
	MissingResults  int
}

// EvalRunItemResult is one terminal placeholder result for a snapped eval-run item.
type EvalRunItemResult struct {
	EvalCaseID   string
	Status       string
	Verdict      string
	Detail       string
	Score        float64
	JudgeVersion string
	JudgeOutput  json.RawMessage
	UpdatedAt    time.Time
}

// EvalReportBadCase is one failed or risky eval case carried on the canonical eval report.
type EvalReportBadCase struct {
	EvalCaseID     string
	Title          string
	SourceCaseID   string
	SourceTaskID   string
	SourceReportID string
	TraceID        string
	VersionID      string
	Verdict        string
	Detail         string
	Score          float64
}

// EvalReport is the durable aggregated artifact for one completed eval run.
type EvalReport struct {
	ID              string
	TenantID        string
	RunID           string
	DatasetID       string
	DatasetName     string
	RunStatus       string
	Status          string
	Summary         string
	TotalItems      int
	RecordedResults int
	PassedItems     int
	FailedItems     int
	MissingResults  int
	AverageScore    float64
	JudgeVersion    string
	MetadataJSON    json.RawMessage
	BadCases        []EvalReportBadCase
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ReadyAt         time.Time
}

// RunListFilter constrains eval-run list reads.
type RunListFilter struct {
	TenantID  string
	DatasetID string
	Status    string
	Limit     int
	Offset    int
}

// RunListPage is one eval-run list page.
type RunListPage struct {
	Runs       []EvalRun
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

// CreateDatasetInput is the typed dataset-draft creation request.
type CreateDatasetInput struct {
	TenantID    string
	Name        string
	Description string
	EvalCaseIDs []string
	CreatedBy   string
}

// AddDatasetItemInput is the typed dataset-membership append request.
type AddDatasetItemInput struct {
	TenantID   string
	EvalCaseID string
	AddedBy    string
}

// PublishDatasetInput is the typed dataset publish request.
type PublishDatasetInput struct {
	TenantID    string
	PublishedBy string
}

// CreateRunInput is the typed eval-run kickoff request.
type CreateRunInput struct {
	TenantID  string
	DatasetID string
	CreatedBy string
}

// EvalReportIDFromRunID derives the stable eval-report ID for one eval run.
func EvalReportIDFromRunID(runID string) string {
	return fmt.Sprintf("eval-report-%s", runID)
}
