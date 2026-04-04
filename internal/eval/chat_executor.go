package eval

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appchat "opspilot-go/internal/app/chat"
)

// ChatService defines the narrow chat interface consumed by the eval executor.
type ChatService interface {
	Handle(ctx context.Context, req appchat.ChatRequestEnvelope) (appchat.HandleResult, error)
}

// ChatRunExecutor executes eval runs by sending each case through the chat service.
type ChatRunExecutor struct {
	chat ChatService
	runs *RunService
}

// NewChatRunExecutor constructs a real eval-run executor backed by the chat service.
func NewChatRunExecutor(chat ChatService, runs *RunService) *ChatRunExecutor {
	return &ChatRunExecutor{
		chat: chat,
		runs: runs,
	}
}

// ExecuteRun sends each eval case item through the chat service.
// Errors from individual items are logged but do not abort the run.
func (e *ChatRunExecutor) ExecuteRun(ctx context.Context, run EvalRun) error {
	detail, err := e.runs.GetRunDetail(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("get run detail: %w", err)
	}

	for _, item := range detail.Items {
		userMessage := item.Title
		if userMessage == "" {
			userMessage = "Evaluate case " + item.EvalCaseID
		}

		_, chatErr := e.chat.Handle(ctx, appchat.ChatRequestEnvelope{
			RequestID:   fmt.Sprintf("eval-%s-%s", run.ID, item.EvalCaseID),
			TraceID:     fmt.Sprintf("eval-%s-%s", run.ID, item.EvalCaseID),
			TenantID:    run.TenantID,
			UserID:      "eval-runner",
			Mode:        "eval",
			UserMessage: userMessage,
			RequestedAt: time.Now().UTC(),
		})
		if chatErr != nil {
			slog.Warn("eval case chat execution failed",
				slog.String("run_id", run.ID),
				slog.String("eval_case_id", item.EvalCaseID),
				slog.Any("error", chatErr),
			)
		}
	}

	return nil
}
