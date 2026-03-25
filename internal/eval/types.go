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

// ErrEvalDatasetNotFound identifies missing durable eval dataset records.
var ErrEvalDatasetNotFound = errors.New("eval dataset not found")

// ErrInvalidEvalDataset identifies invalid eval dataset requests.
var ErrInvalidEvalDataset = errors.New("invalid eval dataset")

// ErrInvalidEvalDatasetState identifies invalid dataset lifecycle transitions.
var ErrInvalidEvalDatasetState = errors.New("invalid eval dataset state")

const (
	// DatasetStatusDraft identifies a draft dataset that is not yet active in regression runs.
	DatasetStatusDraft = "draft"
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
