package session

import (
	"context"
	"fmt"
	"sync"
)

type memoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	messages map[string][]Message
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		sessions: make(map[string]Session),
		messages: make(map[string][]Message),
	}
}

func (s *memoryStore) CreateSession(_ context.Context, sess Session) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return sess, nil
}

func (s *memoryStore) CreateMessage(_ context.Context, msg Message) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[msg.SessionID]; !ok {
		return Message{}, fmt.Errorf("session %q not found", msg.SessionID)
	}
	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)
	return msg, nil
}

func (s *memoryStore) ListMessages(_ context.Context, sessionID string) ([]Message, error) {
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
