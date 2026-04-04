package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/session"
)

// SessionStore persists session and message records in PostgreSQL.
type SessionStore struct {
	pool *pgxpool.Pool
}

// NewSessionStore constructs the session repository.
func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

// CreateSession inserts a new session row.
func (s *SessionStore) CreateSession(ctx context.Context, sess session.Session) (session.Session, error) {
	const query = `
INSERT INTO sessions (id, tenant_id, user_id, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, user_id, created_at`

	var out session.Session
	err := s.pool.QueryRow(ctx, query,
		sess.ID, sess.TenantID, sess.UserID, sess.CreatedAt,
	).Scan(&out.ID, &out.TenantID, &out.UserID, &out.CreatedAt)
	if err != nil {
		return session.Session{}, fmt.Errorf("insert session: %w", err)
	}
	return out, nil
}

// CreateMessage inserts a new message row. Returns an error if the session does not exist.
func (s *SessionStore) CreateMessage(ctx context.Context, msg session.Message) (session.Message, error) {
	const query = `
INSERT INTO messages (id, session_id, role, content, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, session_id, role, content, created_at`

	var out session.Message
	err := s.pool.QueryRow(ctx, query,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt,
	).Scan(&out.ID, &out.SessionID, &out.Role, &out.Content, &out.CreatedAt)
	if err != nil {
		return session.Message{}, fmt.Errorf("insert message for session %q: %w", msg.SessionID, err)
	}
	return out, nil
}

// ListMessages returns messages in append order. Returns an error if the session does not exist.
func (s *SessionStore) ListMessages(ctx context.Context, sessionID string) ([]session.Message, error) {
	const query = `
SELECT id, session_id, role, content, created_at
FROM messages
WHERE session_id = $1
ORDER BY created_at ASC, id ASC`

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages for session %q: %w", sessionID, err)
	}
	defer rows.Close()

	var messages []session.Message
	for rows.Next() {
		var msg session.Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	// Distinguish "session not found" from "session exists but has no messages"
	if len(messages) == 0 {
		var exists int
		err := s.pool.QueryRow(ctx, `SELECT 1 FROM sessions WHERE id = $1`, sessionID).Scan(&exists)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session %q not found", sessionID)
		}
		if err != nil {
			return nil, fmt.Errorf("check session existence: %w", err)
		}
	}

	return messages, nil
}
