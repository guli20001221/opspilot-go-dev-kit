package chat

import (
	"context"

	"opspilot-go/internal/session"
)

// SessionService defines the session operations the chat service consumes.
type SessionService interface {
	CreateSession(ctx context.Context, input session.CreateSessionInput) (session.Session, error)
	AppendMessage(ctx context.Context, input session.AppendMessageInput) (session.Message, error)
}

// Service orchestrates the Milestone 1 chat request flow.
type Service struct {
	sessions SessionService
}

// NewService constructs a chat service with the required downstream dependencies.
func NewService(sessions SessionService) *Service {
	return &Service{sessions: sessions}
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

	if _, err := s.sessions.AppendMessage(ctx, session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleAssistant,
		Content:   PlaceholderAssistantResponse,
	}); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		SessionID: sessionID,
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
