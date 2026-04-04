package session

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

var (
	sessionIDSequence atomic.Uint64
	messageIDSequence atomic.Uint64
)

// Store persists session and message records.
type Store interface {
	CreateSession(ctx context.Context, sess Session) (Session, error)
	CreateMessage(ctx context.Context, msg Message) (Message, error)
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)
}

// Service provides session and message persistence.
type Service struct {
	store Store
}

// NewService creates the session service with a memory-backed default store.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore creates the session service with a caller-provided store.
func NewServiceWithStore(store Store) *Service {
	if store == nil {
		store = newMemoryStore()
	}
	return &Service{store: store}
}

// CreateSession creates a new session record.
func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (Session, error) {
	now := time.Now().UTC()
	sess := Session{
		ID:        newSessionID(now),
		TenantID:  input.TenantID,
		UserID:    input.UserID,
		CreatedAt: now,
	}
	return s.store.CreateSession(ctx, sess)
}

// AppendMessage appends a message to an existing session.
func (s *Service) AppendMessage(ctx context.Context, input AppendMessageInput) (Message, error) {
	now := time.Now().UTC()
	msg := Message{
		ID:        newMessageID(now),
		SessionID: input.SessionID,
		Role:      input.Role,
		Content:   input.Content,
		CreatedAt: now,
	}
	return s.store.CreateMessage(ctx, msg)
}

// ListMessages returns messages in append order for the given session.
func (s *Service) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	return s.store.ListMessages(ctx, sessionID)
}

func newSessionID(now time.Time) string {
	return fmt.Sprintf("sess-%d-%d", now.UnixNano(), sessionIDSequence.Add(1))
}

func newMessageID(now time.Time) string {
	return fmt.Sprintf("msg-%d-%d", now.UnixNano(), messageIDSequence.Add(1))
}
