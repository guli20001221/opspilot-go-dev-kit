package chat

import (
	"context"

	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/session"
)

// SessionService defines the session operations the chat service consumes.
type SessionService interface {
	CreateSession(ctx context.Context, input session.CreateSessionInput) (session.Session, error)
	AppendMessage(ctx context.Context, input session.AppendMessageInput) (session.Message, error)
	ListMessages(ctx context.Context, sessionID string) ([]session.Message, error)
}

// Service orchestrates the Milestone 1 chat request flow.
type Service struct {
	sessions SessionService
	contexts *contextengine.Service
}

// NewService constructs a chat service with the required downstream dependencies.
func NewService(sessions SessionService) *Service {
	return &Service{
		sessions: sessions,
		contexts: contextengine.NewService(contextengine.Config{}),
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

	if _, err := s.sessions.AppendMessage(ctx, session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleAssistant,
		Content:   PlaceholderAssistantResponse,
	}); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		SessionID: sessionID,
		Context:   assembledContext,
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
