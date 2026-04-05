package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	agentcritic "opspilot-go/internal/agent/critic"
	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/observability/metrics"
	"opspilot-go/internal/observability/tracing"
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
	sessions   SessionService
	contexts   *contextengine.Service
	critic     *agentcritic.Service
	planner    *planner.Service
	retrieval  retrieval.Searcher
	reranker   retrieval.Reranker
	compressor *retrieval.ContextualCompressor
	crag       *retrieval.CRAGFilter
	hyde       *retrieval.HyDERewriter
	tools     *agenttool.Service
	registry  *toolregistry.Registry
	workflows *workflow.Service
	llm       llm.Provider
	metrics   *metrics.Instruments
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
// Metrics instruments are auto-created; use NewServiceWithMetrics for explicit injection.
func NewServiceWithLLM(sessions SessionService, workflows *workflow.Service, registry *toolregistry.Registry, searcher retrieval.Searcher, provider llm.Provider) *Service {
	return NewServiceWithMetrics(sessions, workflows, registry, searcher, provider, nil)
}

// NewServiceWithMetrics constructs a chat service with all dependencies including metrics.
// Pass nil for metrics to use auto-created instruments (or no-op in tests).
func NewServiceWithMetrics(sessions SessionService, workflows *workflow.Service, registry *toolregistry.Registry, searcher retrieval.Searcher, provider llm.Provider, m *metrics.Instruments) *Service {
	if workflows == nil {
		workflows = workflow.NewService()
	}
	if registry == nil {
		registry = toolregistry.NewDefaultRegistry()
	}
	if searcher == nil {
		searcher = retrieval.NewService(nil)
	}

	var reranker retrieval.Reranker
	if provider != nil {
		if _, isPlaceholder := provider.(*llm.PlaceholderProvider); !isPlaceholder {
			reranker = retrieval.NewLLMReranker(provider)
		}
	}
	if reranker == nil {
		reranker = &retrieval.NoopReranker{}
	}

	// Context engine with keyword importance scoring by default.
	// Use NewServiceWithDependencies with an EmbeddingImportanceScorer for
	// embedding-based scoring when an embedder is available.
	var scorer contextengine.ImportanceScorer = contextengine.KeywordImportanceScorer{}

	return &Service{
		sessions:  sessions,
		contexts:  contextengine.NewServiceWithDependencies(contextengine.Config{}, nil, scorer),
		critic:    agentcritic.NewServiceWithLLM(provider),
		planner:   planner.NewServiceWithLLM(provider),
		retrieval:  searcher,
		reranker:   reranker,
		compressor: retrieval.NewContextualCompressor(provider),
		crag:       retrieval.NewCRAGFilter(provider),
		hyde:       retrieval.NewHyDERewriter(provider),
		tools:     agenttool.NewService(registry),
		registry:  registry,
		workflows: workflows,
		llm:       provider,
		metrics:   m,
	}
}

// Handle persists the user and assistant turns and returns the ordered SSE events.
func (s *Service) Handle(ctx context.Context, req ChatRequestEnvelope) (HandleResult, error) {
	ctx, span := tracing.StartSpan(ctx, "chat.handle",
		tracing.AttrRequestID.String(req.RequestID),
		tracing.AttrTenantID.String(req.TenantID),
	)
	defer span.End()

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
		UserMessage: req.UserMessage,
		RecentTurns: toTurns(recentMessages),
	})
	if err != nil {
		return HandleResult{}, err
	}

	planStart := time.Now()
	plan, err := s.planner.Plan(ctx, planner.PlanInput{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		TenantID:       req.TenantID,
		SessionID:      sessionID,
		Mode:           req.Mode,
		UserMessage:    req.UserMessage,
		Context:        assembledContext.Planner,
		AvailableTools: toPlannerToolDescriptors(s.registry.List()),
		TenantPolicy:   req.TenantPolicy,
	})
	if err != nil {
		return HandleResult{}, err
	}
	s.metrics.RecordPlannerLatency(ctx, time.Since(planStart), plan.Intent, plan.Source, req.TenantID)

	retrievalResult := retrieval.RetrievalResult{}
	if plan.RequiresRetrieval {
		retrievalStart := time.Now()
		// HyDE: generate a hypothetical document to improve semantic matching
		hydeQuery := req.UserMessage
		if s.hyde != nil {
			if rewritten := s.hyde.Rewrite(ctx, req.UserMessage); rewritten != "" {
				hydeQuery = rewritten
			}
		}

		retrievalResult, err = s.retrieval.Search(ctx, retrieval.RetrievalRequest{
			RequestID:      req.RequestID,
			TraceID:        req.TraceID,
			TenantID:       req.TenantID,
			SessionID:      sessionID,
			PlanID:         plan.PlanID,
			QueryText:      req.UserMessage,
			RewrittenQuery: hydeQuery,
		})
		if err != nil {
			return HandleResult{}, err
		}
		// Re-rank for precision using LLM-based cross-encoder scoring
		if s.reranker != nil {
			reranked, rerankErr := s.reranker.Rerank(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
			if rerankErr != nil {
				slog.Warn("reranking failed, using original order",
					slog.String("request_id", req.RequestID),
					slog.Any("error", rerankErr),
				)
			} else {
				retrievalResult.EvidenceBlocks = reranked
			}
		}
		// Contextual Compression: extract only query-relevant content from each passage
		if s.compressor != nil {
			retrievalResult.EvidenceBlocks = s.compressor.Compress(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
		}
		// CRAG: validate relevance and discard irrelevant passages
		if s.crag != nil {
			filtered, cragStats := s.crag.Filter(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
			if cragStats.Total > 0 {
				slog.Info("crag filter applied",
					slog.String("request_id", req.RequestID),
					slog.Int("relevant", cragStats.Relevant),
					slog.Int("ambiguous", cragStats.Ambiguous),
					slog.Int("irrelevant", cragStats.Irrelevant),
				)
			}
			retrievalResult.EvidenceBlocks = filtered
		}
		// Apply lost-in-the-middle reordering for optimal LLM context placement
		retrievalResult.EvidenceBlocks = retrieval.ReorderLostInTheMiddle(retrievalResult.EvidenceBlocks)
		s.metrics.RecordRetrievalLatency(ctx, time.Since(retrievalStart), len(retrievalResult.EvidenceBlocks), req.TenantID)
	}

	isEvalMode := req.Mode == "eval"

	toolResults := make([]agenttool.ToolResult, 0, len(plan.Steps))
	toolInvocations := make([]agenttool.ToolInvocation, 0, len(plan.Steps))
	var replanCount int
	activePlan := plan // activePlan tracks the current plan (may be revised by replanning)
	if !isEvalMode {
		var activePlanResult planner.ExecutionPlan
		toolResults, toolInvocations, replanCount, activePlanResult, err = s.executeToolSteps(ctx, req, sessionID, plan)
		if err != nil {
			return HandleResult{}, err
		}
		activePlan = activePlanResult
	}

	// Post-replan retrieval: if the revised plan requires retrieval but the
	// original plan did not (or retrieval returned no evidence), run the
	// retrieval pipeline now so LLM completion has grounding context.
	if replanCount > 0 && activePlan.RequiresRetrieval && len(retrievalResult.EvidenceBlocks) == 0 {
		retrievalStart := time.Now()
		hydeQuery := req.UserMessage
		if s.hyde != nil {
			if rewritten := s.hyde.Rewrite(ctx, req.UserMessage); rewritten != "" {
				hydeQuery = rewritten
			}
		}
		retrievalResult, err = s.retrieval.Search(ctx, retrieval.RetrievalRequest{
			RequestID:      req.RequestID,
			TraceID:        req.TraceID,
			TenantID:       req.TenantID,
			SessionID:      sessionID,
			PlanID:         activePlan.PlanID,
			QueryText:      req.UserMessage,
			RewrittenQuery: hydeQuery,
		})
		if err != nil {
			slog.Warn("post-replan retrieval failed, continuing without evidence",
				slog.String("request_id", req.RequestID),
				slog.Any("error", err),
			)
			retrievalResult = retrieval.RetrievalResult{}
		} else {
			if s.reranker != nil {
				reranked, rerankErr := s.reranker.Rerank(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
				if rerankErr == nil {
					retrievalResult.EvidenceBlocks = reranked
				}
			}
			if s.compressor != nil {
				retrievalResult.EvidenceBlocks = s.compressor.Compress(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
			}
			if s.crag != nil {
				filtered, _ := s.crag.Filter(ctx, req.UserMessage, retrievalResult.EvidenceBlocks)
				retrievalResult.EvidenceBlocks = filtered
			}
			retrievalResult.EvidenceBlocks = retrieval.ReorderLostInTheMiddle(retrievalResult.EvidenceBlocks)
		}
		s.metrics.RecordRetrievalLatency(ctx, time.Since(retrievalStart), len(retrievalResult.EvidenceBlocks), req.TenantID)
	}

	// Rebuild context with retrieval evidence and tool results for the LLM completion.
	// This ensures the context engine's stage-aware assembly, summarization, and
	// dynamic importance scoring are applied to the final answer — not just planning.
	completionContext, _ := s.contexts.Build(ctx, contextengine.BuildInput{
		RequestID:   req.RequestID,
		SessionID:   sessionID,
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Mode:        req.Mode,
		UserMessage: req.UserMessage,
		RecentTurns: toTurns(recentMessages),
		RetrievalResults: toEvidenceSnippets(retrievalResult.EvidenceBlocks),
		ToolResults:      toToolResultSnippets(toolResults),
	})

	// Generate assistant response via LLM (or placeholder fallback)
	// Uses streaming when the provider supports it and OnToken callback is set.
	assistantContent := PlaceholderAssistantResponse
	if s.llm != nil {
		llmStart := time.Now()
		completionReq := s.buildCompletionRequestFromContext(req, completionContext.Critic)

		var resp llm.CompletionResponse
		var llmErr error

		if streamer, ok := s.llm.(llm.StreamingProvider); ok && req.OnToken != nil {
			resp, llmErr = streamer.StreamComplete(ctx, completionReq, req.OnToken)
		} else {
			resp, llmErr = s.llm.Complete(ctx, completionReq)
		}

		if llmErr != nil {
			slog.Warn("llm completion failed, falling back to placeholder",
				slog.String("request_id", req.RequestID),
				slog.Any("error", llmErr),
			)
		} else if resp.Content != "" {
			assistantContent = resp.Content
		}
		s.metrics.RecordLLMCall(ctx, time.Since(llmStart), resp.Model, resp.PromptTokens, resp.OutputTokens)
	}

	criticStart := time.Now()
	criticVerdict, err := s.critic.Review(ctx, agentcritic.CriticInput{
		Plan:        activePlan,
		Retrieval:   &retrievalResult,
		ToolResults: toolResults,
		DraftAnswer: assistantContent,
	})
	if err != nil {
		return HandleResult{}, err
	}
	s.metrics.RecordCriticVerdict(ctx, time.Since(criticStart), criticVerdict.Verdict, criticVerdict.Source)

	if replanCount > 0 {
		s.metrics.RecordReplan(ctx, "tool_failure")
	}

	var promotedTask *workflow.Task
	if !isEvalMode && (activePlan.RequiresWorkflow || criticVerdict.Verdict == agentcritic.VerdictPromoteWorkflow) {
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
		Plan:         activePlan,
		Retrieval:    retrievalResult,
		ToolResults:  toolResults,
		Critic:       criticVerdict,
		PromotedTask: promotedTask,
		ReplanCount:  replanCount,
		Events:       buildEvents(req, sessionID, activePlan, retrievalResult, toolResults, promotedTask, assistantContent),
	}, nil
}

const maxReplanAttempts = 1

// executeToolSteps runs tool steps from the plan, with dynamic replanning on failure.
// Returns all tool results, invocations, the replan count, the active plan (revised if
// replanning occurred, original otherwise), and any error.
func (s *Service) executeToolSteps(
	ctx context.Context,
	req ChatRequestEnvelope,
	sessionID string,
	plan planner.ExecutionPlan,
) ([]agenttool.ToolResult, []agenttool.ToolInvocation, int, planner.ExecutionPlan, error) {
	var (
		results     []agenttool.ToolResult
		invocations []agenttool.ToolInvocation
		replanCount int
	)

	for _, step := range plan.Steps {
		if step.Kind != planner.StepKindTool {
			continue
		}

		var args json.RawMessage
		var err error
		if len(step.ToolArguments) > 0 {
			args = step.ToolArguments
		} else if !step.ReadOnly {
			// Write (side-effecting) tools MUST have structured arguments from
			// the planner. Heuristic fallback is not safe for write operations
			// because it dumps raw user text into tool parameters.
			slog.Warn("write tool has no structured arguments, refusing execution",
				slog.String("request_id", req.RequestID),
				slog.String("tool_name", step.ToolName),
				slog.String("plan_source", plan.Source),
			)
			return nil, nil, replanCount, plan, fmt.Errorf("write tool %q requires structured tool_arguments from planner", step.ToolName)
		} else {
			// Read-only tools: heuristic fallback is acceptable
			slog.Debug("read-only tool using heuristic argument fallback",
				slog.String("request_id", req.RequestID),
				slog.String("tool_name", step.ToolName),
				slog.String("plan_source", plan.Source),
			)
			args, err = buildToolArguments(step.ToolName, req.UserMessage)
			if err != nil {
				return nil, nil, replanCount, plan, err
			}
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
		invocations = append(invocations, invocation)

		toolStart := time.Now()
		toolResult, execErr := s.tools.Execute(ctx, invocation)
		if execErr != nil {
			s.metrics.RecordToolExecution(ctx, time.Since(toolStart), step.ToolName, agenttool.StatusFailed, req.TenantID)
			// Tool execution hard-failed — attempt dynamic replanning
			if replanCount < maxReplanAttempts && plan.Source != planner.PlanSourceKeyword {
				slog.Warn("tool execution failed, attempting replan",
					slog.String("request_id", req.RequestID),
					slog.String("tool_name", step.ToolName),
					slog.Any("error", execErr),
				)

				executedSteps := buildExecutedSteps(results)
				executedSteps = append(executedSteps, planner.ExecutedStep{
					StepID:   step.StepID,
					Kind:     step.Kind,
					ToolName: step.ToolName,
					Status:   agenttool.StatusFailed,
					Summary:  execErr.Error(),
				})

				replanInput := planner.ReplanInput{
					OriginalPlan:  plan,
					ExecutedSteps: executedSteps,
					Input: planner.PlanInput{
						RequestID:      req.RequestID,
						TenantID:       req.TenantID,
						SessionID:      sessionID,
						Mode:           req.Mode,
						UserMessage:    req.UserMessage,
						AvailableTools: toPlannerToolDescriptors(s.registry.List()),
						TenantPolicy:   req.TenantPolicy,
					},
					ReplanReason: fmt.Sprintf("tool %s failed: %s", step.ToolName, execErr.Error()),
				}

				revisedPlan, replanErr := s.planner.Replan(ctx, replanInput)
				if replanErr != nil {
					slog.Warn("replan failed, propagating original error",
						slog.String("request_id", req.RequestID),
						slog.Any("replan_error", replanErr),
					)
					return nil, nil, replanCount, plan, execErr
				}

				replanCount++
				// Record the failed tool attempt so it appears in audit trail / SSE events
				results = append(results, agenttool.ToolResult{
					ToolCallID:    fmt.Sprintf("toolcall-%s-%s", plan.PlanID, step.StepID),
					ToolName:      step.ToolName,
					Status:        agenttool.StatusFailed,
					OutputSummary: execErr.Error(),
					AuditRef:      fmt.Sprintf("audit-%s-%s", plan.PlanID, step.StepID),
				})
				// Execute the revised plan's tool steps (no further replanning)
				for _, rStep := range revisedPlan.Steps {
					if rStep.Kind != planner.StepKindTool {
						continue
					}
					var rArgs json.RawMessage
					if len(rStep.ToolArguments) > 0 {
						rArgs = rStep.ToolArguments
					} else if !rStep.ReadOnly {
						// Write-tool safety: same boundary as the primary path
						slog.Warn("replan write tool has no structured arguments, refusing execution",
							slog.String("request_id", req.RequestID),
							slog.String("tool_name", rStep.ToolName),
						)
						return nil, nil, replanCount, revisedPlan, fmt.Errorf("write tool %q requires structured tool_arguments from planner", rStep.ToolName)
					} else {
						rArgs, err = buildToolArguments(rStep.ToolName, req.UserMessage)
						if err != nil {
							return nil, nil, replanCount, revisedPlan, err
						}
					}
					rInvocation := agenttool.ToolInvocation{
						RequestID:        req.RequestID,
						TraceID:          req.TraceID,
						TenantID:         req.TenantID,
						SessionID:        sessionID,
						PlanID:           revisedPlan.PlanID,
						StepID:           rStep.StepID,
						ToolName:         rStep.ToolName,
						ActionClass:      actionClassForStep(rStep),
						RequiresApproval: rStep.NeedsApproval,
						Arguments:        rArgs,
					}
					invocations = append(invocations, rInvocation)
					rResult, rErr := s.tools.Execute(ctx, rInvocation)
					if rErr != nil {
						return nil, nil, replanCount, revisedPlan, rErr
					}
					results = append(results, rResult)
				}
				return results, invocations, replanCount, revisedPlan, nil
			}
			return nil, nil, replanCount, plan, execErr
		}
		s.metrics.RecordToolExecution(ctx, time.Since(toolStart), step.ToolName, toolResult.Status, req.TenantID)
		results = append(results, toolResult)
	}

	return results, invocations, replanCount, plan, nil
}

// buildExecutedSteps converts completed tool results into planner ExecutedStep records.
func buildExecutedSteps(results []agenttool.ToolResult) []planner.ExecutedStep {
	steps := make([]planner.ExecutedStep, 0, len(results))
	for _, r := range results {
		steps = append(steps, planner.ExecutedStep{
			StepID:   r.ToolCallID,
			Kind:     planner.StepKindTool,
			ToolName: r.ToolName,
			Status:   r.Status,
			Summary:  r.OutputSummary,
		})
	}
	return steps
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
		params := make([]planner.ToolParameterDesc, 0, len(def.Parameters))
		for _, p := range def.Parameters {
			params = append(params, planner.ToolParameterDesc{
				Name:        p.Name,
				Type:        p.Type,
				Required:    p.Required,
				Description: p.Description,
			})
		}
		out = append(out, planner.ToolDescriptor{
			Name:             def.Name,
			Description:      def.Description,
			ReadOnly:         def.ReadOnly,
			RequiresApproval: def.RequiresApproval,
			AsyncOnly:        def.AsyncOnly,
			Parameters:       params,
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

// buildCompletionRequestFromContext assembles the LLM completion request from
// the context engine's critic context. This ensures summarization, importance
// scoring, and budget eviction are applied to the final answer — not just planning.
func (s *Service) buildCompletionRequestFromContext(
	req ChatRequestEnvelope,
	criticCtx contextengine.CriticContext,
) llm.CompletionRequest {
	var systemParts []string
	systemParts = append(systemParts, chatSystemPrompt)

	// Assemble system prompt from context engine blocks (already budget-managed)
	for _, block := range criticCtx.Blocks {
		switch block.Kind {
		case contextengine.BlockKindRetrievalEvidence:
			systemParts = append(systemParts, "\n\nEvidence: "+block.Content)
		case contextengine.BlockKindToolResult:
			systemParts = append(systemParts, "\n\nTool result: "+block.Content)
		case contextengine.BlockKindSessionSummary:
			systemParts = append(systemParts, "\n\nConversation summary: "+block.Content)
		case contextengine.BlockKindUserProfile:
			systemParts = append(systemParts, "\n\nUser context: "+block.Content)
		}
	}

	// Recent turns as messages (from critic context or direct)
	var messages []llm.Message
	for _, block := range criticCtx.Blocks {
		if block.Kind == contextengine.BlockKindRecentTurns {
			// The block content is formatted turns — use it as a single user context message
			messages = append(messages, llm.Message{Role: "user", Content: req.UserMessage})
			break
		}
	}
	if len(messages) == 0 {
		messages = []llm.Message{{Role: "user", Content: req.UserMessage}}
	}

	return llm.CompletionRequest{
		SystemPrompt: strings.Join(systemParts, ""),
		Messages:     messages,
		MaxTokens:    1024,
	}
}

func toEvidenceSnippets(blocks []retrieval.EvidenceBlock) []contextengine.EvidenceSnippet {
	snippets := make([]contextengine.EvidenceSnippet, 0, len(blocks))
	for _, b := range blocks {
		snippets = append(snippets, contextengine.EvidenceSnippet{
			SourceTitle:   b.SourceTitle,
			Snippet:       b.Snippet,
			CitationLabel: b.CitationLabel,
			Score:         b.Score,
		})
	}
	return snippets
}

func toToolResultSnippets(results []agenttool.ToolResult) []contextengine.ToolResultSnippet {
	snippets := make([]contextengine.ToolResultSnippet, 0, len(results))
	for _, r := range results {
		snippets = append(snippets, contextengine.ToolResultSnippet{
			ToolName:      r.ToolName,
			Status:        r.Status,
			OutputSummary: r.OutputSummary,
		})
	}
	return snippets
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
				"source":             plan.Source,
				"prompt_version":     plan.PromptVersion,
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
