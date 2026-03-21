package chat

import (
	"time"

	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/retrieval"
)

// PlaceholderAssistantResponse is the fixed M1 assistant content used before agent runtime lands.
const PlaceholderAssistantResponse = "Milestone 1 placeholder response."

// ChatRequestEnvelope is the typed Milestone 1 chat request DTO at the application boundary.
type ChatRequestEnvelope struct {
	RequestID       string
	TraceID         string
	TenantID        string
	UserID          string
	SessionID       string
	Mode            string
	UserMessage     string
	AttachmentRefs  []string
	ClientRequestID string
	RequestedAt     time.Time
}

// StreamEvent represents one server-sent event payload produced by the chat application service.
type StreamEvent struct {
	Name string
	Data map[string]string
}

// HandleResult contains the persisted session identifier and ordered stream events.
type HandleResult struct {
	SessionID   string
	Context     contextengine.BuildResult
	Plan        planner.ExecutionPlan
	Retrieval   retrieval.RetrievalResult
	ToolResults []agenttool.ToolResult
	Events      []StreamEvent
}
