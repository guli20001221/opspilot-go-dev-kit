package version

import (
	"errors"
	"time"
)

const (
	// DefaultVersionID identifies the current built-in runtime version snapshot.
	DefaultVersionID = "version-skeleton-2026-03-24"
)

// ErrVersionNotFound identifies missing durable version records.
var ErrVersionNotFound = errors.New("version not found")

// Version is the durable operator-facing runtime version snapshot.
type Version struct {
	ID                  string
	RuntimeVersion      string
	Provider            string
	Model               string
	PromptBundle        string
	PlannerVersion      string
	RetrievalVersion    string
	ToolRegistryVersion string
	CriticVersion       string
	WorkflowVersion     string
	Notes               string
	CreatedAt           time.Time
}

// ListFilter constrains durable version list reads.
type ListFilter struct {
	Limit  int
	Offset int
}

// ListPage is a single version list page.
type ListPage struct {
	Versions   []Version
	HasMore    bool
	NextOffset int
}
