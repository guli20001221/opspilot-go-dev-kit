package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Service provides in-memory session and message persistence for the M1 skeleton.
type Service struct {
	mu       sync.RWMutex
	sessions map[string]Session
	messages map[string][]Message
}

// NewService creates the minimum session service implementation.
func NewService() *Service {
	return &Service{
		sessions: make(map[string]Session),
		messages: make(map[string][]Message),
	}
}

// CreateSession creates a new session record.
func (s *Service) CreateSession(_ context.Context, input CreateSessionInput) (Session, error) {
	now := time.Now().UTC()
	session := Session{
		ID:        newID(now),
		TenantID:  input.TenantID,
		UserID:    input.UserID,
		CreatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session

	return session, nil
}

// AppendMessage appends a message to an existing session.
func (s *Service) AppendMessage(_ context.Context, input AppendMessageInput) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[input.SessionID]; !ok {
		return Message{}, fmt.Errorf("session %q not found", input.SessionID)
	}

	now := time.Now().UTC()
	message := Message{
		ID:        newID(now),
		SessionID: input.SessionID,
		Role:      input.Role,
		Content:   input.Content,
		CreatedAt: now,
	}
	s.messages[input.SessionID] = append(s.messages[input.SessionID], message)

	return message, nil
}

// ListMessages returns messages in append order for the given session.
func (s *Service) ListMessages(_ context.Context, sessionID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.sessions[sessionID]; !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}

	stored := s.messages[sessionID]
	out := make([]Message, len(stored))
	copy(out, stored)

	return out, nil
}

func newID(now time.Time) string {
	return now.Format("20060102150405.000000000")
}
