package session

import "time"

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Session represents a user conversation container.
type Session struct {
	ID        string
	TenantID  string
	UserID    string
	CreatedAt time.Time
}

// Message represents a persisted turn inside a session.
type Message struct {
	ID        string
	SessionID string
	Role      string
	Content   string
	CreatedAt time.Time
}

// CreateSessionInput contains the minimum fields required to open a session.
type CreateSessionInput struct {
	TenantID string
	UserID   string
}

// AppendMessageInput contains the minimum fields required to append a turn.
type AppendMessageInput struct {
	SessionID string
	Role      string
	Content   string
}
