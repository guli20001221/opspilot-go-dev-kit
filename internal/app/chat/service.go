package chat

import (
	"context"
	"encoding/json"

	agentcritic "opspilot-go/internal/agent/critic"
	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/contextengine"
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
	retrieval *retrieval.Service
	tools     *agenttool.Service
	registry  *toolregistry.Registry
	workflows *workflow.Service
}

// NewService constructs a chat service with the required downstream dependencies.
func NewService(sessions SessionService) *Service {
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:             "ticket_search",
		ActionClass:      agenttool.ActionClassRead,
		ReadOnly:         true,
		RequiresApproval: false,
		StubResponse: map[string]any{
			"matches": []map[string]string{
				{"ticket_id": "INC-100", "summary": "database incident"},
			},
		},
	})
	registry.Register(toolregistry.Definition{
		Name:             "ticket_comment_create",
		ActionClass:      agenttool.ActionClassWrite,
		ReadOnly:         false,
		RequiresApproval: true,
	})

	return &Service{
		sessions:  sessions,
		contexts:  contextengine.NewService(contextengine.Config{}),
		critic:    agentcritic.NewService(),
		planner:   planner.NewService(),
		retrieval: retrieval.NewService(nil),
		tools:     agenttool.NewService(registry),
		registry:  registry,
		workflows: workflow.NewService(),
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
	for _, step := range plan.Steps {
		if step.Kind != planner.StepKindTool {
			continue
		}

		args, err := json.Marshal(map[string]string{"query": req.UserMessage})
		if err != nil {
			return HandleResult{}, err
		}

		toolResult, err := s.tools.Execute(ctx, agenttool.ToolInvocation{
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
		})
		if err != nil {
			return HandleResult{}, err
		}
		toolResults = append(toolResults, toolResult)
	}

	criticVerdict, err := s.critic.Review(ctx, agentcritic.CriticInput{
		Plan:        plan,
		Retrieval:   &retrievalResult,
		ToolResults: toolResults,
		DraftAnswer: PlaceholderAssistantResponse,
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

		task, err := s.workflows.Promote(ctx, workflow.PromoteRequest{
			RequestID:        req.RequestID,
			TenantID:         req.TenantID,
			SessionID:        sessionID,
			TaskType:         taskType,
			Reason:           reason,
			RequiresApproval: requiresApproval,
		})
		if err != nil {
			return HandleResult{}, err
		}
		promotedTask = &task
	}

	if _, err := s.sessions.AppendMessage(ctx, session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleAssistant,
		Content:   PlaceholderAssistantResponse,
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
		Events: []StreamEvent{
			{
				Name: "meta",
				Data: map[string]string{
					"request_id": req.RequestID,
					"trace_id":   req.TraceID,
					"session_id": sessionID,
				},
			},
			{
				Name: "state",
				Data: map[string]string{
					"state": "completed",
				},
			},
			{
				Name: "done",
				Data: map[string]string{
					"session_id": sessionID,
					"content":    PlaceholderAssistantResponse,
				},
			},
		},
	}, nil
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
