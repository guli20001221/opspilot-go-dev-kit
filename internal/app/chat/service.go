package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	agentcritic "opspilot-go/internal/agent/critic"
	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
	toolregistry "opspilot-go/internal/tools/registry"
	"opspilot-go/internal/workflow"
)

// SessionService defines the session operations the chat service consumes.
type SessionService interface {
	CreateSession(ctx context.Context, input session.CreateSessionInput) (session.Session, error)
	AppendMessage(ctx context.Context, input session.AppendMessageInput) (session.Message, error)
	ListMessages(ctx context.Context, sessionID string) ([]session.Message, error)
}

// Service orchestrates the Milestone 1 chat request flow.
type Service struct {
	sessions  SessionService
	contexts  *contextengine.Service
	critic    *agentcritic.Service
	planner   *planner.Service
	retrieval retrieval.Searcher
	tools     *agenttool.Service
	registry  *toolregistry.Registry
	workflows *workflow.Service
	llm       llm.Provider
}

// NewService constructs a chat service with the required downstream dependencies.
func NewService(sessions SessionService) *Service {
	return NewServiceWithRegistry(sessions, workflow.NewService(), toolregistry.NewDefaultRegistry())
}

// NewServiceWithWorkflow constructs a chat service with a caller-provided workflow service.
func NewServiceWithWorkflow(sessions SessionService, workflows *workflow.Service) *Service {
	return NewServiceWithRegistry(sessions, workflows, toolregistry.NewDefaultRegistry())
}

// NewServiceWithRegistry constructs a chat service with caller-provided workflow
// service and tool registry.
func NewServiceWithRegistry(sessions SessionService, workflows *workflow.Service, registry *toolregistry.Registry) *Service {
	return NewServiceWithDependencies(sessions, workflows, registry, nil)
}

// NewServiceWithDependencies constructs a chat service with all optional dependencies.
func NewServiceWithDependencies(sessions SessionService, workflows *workflow.Service, registry *toolregistry.Registry, searcher retrieval.Searcher) *Service {
	return NewServiceWithLLM(sessions, workflows, registry, searcher, nil)
}

// NewServiceWithLLM constructs a chat service with an LLM provider for response generation.
func NewServiceWithLLM(sessions SessionService, workflows *workflow.Service, registry *toolregistry.Registry, searcher retrieval.Searcher, provider llm.Provider) *Service {
	if workflows == nil {
		workflows = workflow.NewService()
	}
	if registry == nil {
		registry = toolregistry.NewDefaultRegistry()
	}
	if searcher == nil {
		searcher = retrieval.NewService(nil)
	}

	return &Service{
		sessions:  sessions,
		contexts:  contextengine.NewService(contextengine.Config{}),
		critic:    agentcritic.NewService(),
		planner:   planner.NewService(),
		retrieval: searcher,
		tools:     agenttool.NewService(registry),
		registry:  registry,
		workflows: workflows,
		llm:       provider,
	}
}

// Handle persists the user and assistant turns and returns the ordered SSE events.
func (s *Service) Handle(ctx context.Context, req ChatRequestEnvelope) (HandleResult, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		created, err := s.sessions.CreateSession(ctx, session.CreateSessionInput{
			TenantID: req.TenantID,
			UserID:   req.UserID,
		})
		if err != nil {
			return HandleResult{}, err
		}
		sessionID = created.ID
	}

	if _, err := s.sessions.AppendMessage(ctx, session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleUser,
		Content:   req.UserMessage,
	}); err != nil {
		return HandleResult{}, err
	}

	recentMessages, err := s.sessions.ListMessages(ctx, sessionID)
	if err != nil {
		return HandleResult{}, err
	}

	assembledContext, err := s.contexts.Build(ctx, contextengine.BuildInput{
		RequestID:   req.RequestID,
		SessionID:   sessionID,
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Mode:        req.Mode,
		RecentTurns: toTurns(recentMessages),
	})
	if err != nil {
		return HandleResult{}, err
	}

	plan, err := s.planner.Plan(ctx, planner.PlanInput{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		TenantID:       req.TenantID,
		SessionID:      sessionID,
		Mode:           req.Mode,
		UserMessage:    req.UserMessage,
		Context:        assembledContext.Planner,
		AvailableTools: toPlannerToolDescriptors(s.registry.List()),
	})
	if err != nil {
		return HandleResult{}, err
	}

	retrievalResult := retrieval.RetrievalResult{}
	if plan.RequiresRetrieval {
		retrievalResult, err = s.retrieval.Search(ctx, retrieval.RetrievalRequest{
			RequestID: req.RequestID,
			TraceID:   req.TraceID,
			TenantID:  req.TenantID,
			SessionID: sessionID,
			PlanID:    plan.PlanID,
			QueryText: req.UserMessage,
		})
		if err != nil {
			return HandleResult{}, err
		}
	}

	toolResults := make([]agenttool.ToolResult, 0, len(plan.Steps))
	toolInvocations := make([]agenttool.ToolInvocation, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		if step.Kind != planner.StepKindTool {
			continue
		}

		args, err := buildToolArguments(step.ToolName, req.UserMessage)
		if err != nil {
			return HandleResult{}, err
		}

		invocation := agenttool.ToolInvocation{
			RequestID:        req.RequestID,
			TraceID:          req.TraceID,
			TenantID:         req.TenantID,
			SessionID:        sessionID,
			PlanID:           plan.PlanID,
			StepID:           step.StepID,
			ToolName:         step.ToolName,
			ActionClass:      actionClassForStep(step),
			RequiresApproval: step.NeedsApproval,
			Arguments:        args,
		}
		toolInvocations = append(toolInvocations, invocation)

		toolResult, err := s.tools.Execute(ctx, invocation)
		if err != nil {
			return HandleResult{}, err
		}
		toolResults = append(toolResults, toolResult)
	}

	// Generate assistant response via LLM (or placeholder fallback)
	assistantContent := PlaceholderAssistantResponse
	if s.llm != nil {
		completionReq := s.buildCompletionRequest(req, recentMessages, retrievalResult, toolResults)
		resp, llmErr := s.llm.Complete(ctx, completionReq)
		if llmErr != nil {
			slog.Warn("llm completion failed, falling back to placeholder",
				slog.String("request_id", req.RequestID),
				slog.Any("error", llmErr),
			)
		} else if resp.Content != "" {
			assistantContent = resp.Content
		}
	}

	criticVerdict, err := s.critic.Review(ctx, agentcritic.CriticInput{
		Plan:        plan,
		Retrieval:   &retrievalResult,
		ToolResults: toolResults,
		DraftAnswer: assistantContent,
	})
	if err != nil {
		return HandleResult{}, err
	}

	var promotedTask *workflow.Task
	if plan.RequiresWorkflow || criticVerdict.Verdict == agentcritic.VerdictPromoteWorkflow {
		taskType := workflow.TaskTypeReportGeneration
		reason := workflow.PromotionReasonWorkflowRequired
		requiresApproval := false
		if criticVerdict.Verdict == agentcritic.VerdictPromoteWorkflow {
			taskType = workflow.TaskTypeApprovedToolExecution
			reason = workflow.PromotionReasonApprovalRequired
			requiresApproval = true
		}

		var toolName string
		var toolArguments json.RawMessage
		if requiresApproval {
			if approvalInvocation, ok := firstApprovalInvocation(toolInvocations, toolResults); ok {
				toolName = approvalInvocation.ToolName
				toolArguments = approvalInvocation.Arguments
			}
		}

		task, err := s.workflows.Promote(ctx, workflow.PromoteRequest{
			RequestID:        req.RequestID,
			TenantID:         req.TenantID,
			SessionID:        sessionID,
			TaskType:         taskType,
			Reason:           reason,
			RequiresApproval: requiresApproval,
			ToolName:         toolName,
			ToolArguments:    toolArguments,
		})
		if err != nil {
			return HandleResult{}, err
		}
		promotedTask = &task
	}

	if _, err := s.sessions.AppendMessage(ctx, session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleAssistant,
		Content:   assistantContent,
	}); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		SessionID:    sessionID,
		Context:      assembledContext,
		Plan:         plan,
		Retrieval:    retrievalResult,
		ToolResults:  toolResults,
		Critic:       criticVerdict,
		PromotedTask: promotedTask,
		Events:       buildEvents(req, sessionID, plan, retrievalResult, toolResults, promotedTask, assistantContent),
	}, nil
}

var ticketIDPattern = regexp.MustCompile(`(?i)\b[A-Z]+-\d+\b`)

func buildToolArguments(toolName string, userMessage string) (json.RawMessage, error) {
	switch toolName {
	case "ticket_comment_create":
		payload := map[string]string{
			"comment": userMessage,
		}
		if ticketID := ticketIDPattern.FindString(userMessage); ticketID != "" {
			payload["ticket_id"] = ticketID
		}
		return json.Marshal(payload)
	default:
		return json.Marshal(map[string]string{"query": userMessage})
	}
}

func firstApprovalInvocation(invocations []agenttool.ToolInvocation, results []agenttool.ToolResult) (agenttool.ToolInvocation, bool) {
	for _, invocation := range invocations {
		for _, result := range results {
			if result.ToolName == invocation.ToolName && result.Status == agenttool.StatusApprovalRequired {
				return invocation, true
			}
		}
	}

	return agenttool.ToolInvocation{}, false
}

func toTurns(messages []session.Message) []contextengine.Turn {
	turns := make([]contextengine.Turn, 0, len(messages))
	for _, message := range messages {
		turns = append(turns, contextengine.Turn{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	return turns
}

func toPlannerToolDescriptors(defs []toolregistry.Definition) []planner.ToolDescriptor {
	out := make([]planner.ToolDescriptor, 0, len(defs))
	for _, def := range defs {
		out = append(out, planner.ToolDescriptor{
			Name:             def.Name,
			ReadOnly:         def.ReadOnly,
			RequiresApproval: def.RequiresApproval,
			AsyncOnly:        def.AsyncOnly,
		})
	}

	return out
}

func actionClassForStep(step planner.PlanStep) string {
	if step.ReadOnly {
		return agenttool.ActionClassRead
	}

	return agenttool.ActionClassWrite
}

const chatSystemPrompt = `You are OpsPilot, an enterprise operations assistant. Answer the user's question based on the provided context. If retrieval evidence is available, cite it. If tool results are available, incorporate them. Be concise and accurate. If you don't have enough information to answer, say so clearly.`

func (s *Service) buildCompletionRequest(
	req ChatRequestEnvelope,
	recentMessages []session.Message,
	retrievalResult retrieval.RetrievalResult,
	toolResults []agenttool.ToolResult,
) llm.CompletionRequest {
	var systemParts []string
	systemParts = append(systemParts, chatSystemPrompt)

	if len(retrievalResult.EvidenceBlocks) > 0 {
		systemParts = append(systemParts, "\n\nRetrieved evidence:")
		for _, block := range retrievalResult.EvidenceBlocks {
			systemParts = append(systemParts, "\n"+block.CitationLabel+" "+block.SourceTitle+": "+block.Snippet)
		}
	}

	if len(toolResults) > 0 {
		systemParts = append(systemParts, "\n\nTool results:")
		for _, tr := range toolResults {
			systemParts = append(systemParts, "\n"+tr.ToolName+": "+tr.OutputSummary)
		}
	}

	messages := make([]llm.Message, 0, len(recentMessages))
	for _, msg := range recentMessages {
		messages = append(messages, llm.Message{Role: msg.Role, Content: msg.Content})
	}

	return llm.CompletionRequest{
		SystemPrompt: strings.Join(systemParts, ""),
		Messages:     messages,
		MaxTokens:    1024,
	}
}

func buildEvents(
	req ChatRequestEnvelope,
	sessionID string,
	plan planner.ExecutionPlan,
	retrievalResult retrieval.RetrievalResult,
	toolResults []agenttool.ToolResult,
	promotedTask *workflow.Task,
	assistantContent string,
) []StreamEvent {
	events := []StreamEvent{
		{
			Name: "meta",
			Data: map[string]string{
				"request_id": req.RequestID,
				"trace_id":   req.TraceID,
				"session_id": sessionID,
			},
		},
		{
			Name: "plan",
			Data: map[string]string{
				"plan_id":            plan.PlanID,
				"intent":             plan.Intent,
				"requires_retrieval": strconv.FormatBool(plan.RequiresRetrieval),
				"requires_tool":      strconv.FormatBool(plan.RequiresTool),
				"requires_workflow":  strconv.FormatBool(plan.RequiresWorkflow),
			},
		},
	}

	if plan.RequiresRetrieval {
		events = append(events, StreamEvent{
			Name: "retrieval",
			Data: map[string]string{
				"query_used":     retrievalResult.QueryUsed,
				"evidence_count": strconv.Itoa(len(retrievalResult.EvidenceBlocks)),
			},
		})
	}

	for _, toolResult := range toolResults {
		events = append(events, StreamEvent{
			Name: "tool",
			Data: map[string]string{
				"tool_name": toolResult.ToolName,
				"status":    toolResult.Status,
			},
		})
	}

	if promotedTask != nil {
		events = append(events, StreamEvent{
			Name: "task_promoted",
			Data: map[string]string{
				"task_id": promotedTask.ID,
				"status":  promotedTask.Status,
				"reason":  promotedTask.Reason,
			},
		})
	}

	events = append(events,
		StreamEvent{
			Name: "state",
			Data: map[string]string{
				"state": "completed",
			},
		},
		StreamEvent{
			Name: "done",
			Data: map[string]string{
				"session_id": sessionID,
				"content":    assistantContent,
			},
		},
	)

	return events
}
