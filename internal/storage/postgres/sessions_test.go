package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"opspilot-go/internal/session"
)

func TestSessionStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DELETE FROM messages"); err != nil {
		t.Fatalf("DELETE messages error = %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM sessions"); err != nil {
		t.Fatalf("DELETE sessions error = %v", err)
	}

	store := NewSessionStore(pool)

	// Create session
	now := time.Now().UTC().Truncate(time.Microsecond)
	sess := session.Session{
		ID:        "sess-pg-test-1",
		TenantID:  "tenant-pg",
		UserID:    "user-pg",
		CreatedAt: now,
	}
	created, err := store.CreateSession(ctx, sess)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.ID != sess.ID {
		t.Fatalf("created.ID = %q, want %q", created.ID, sess.ID)
	}
	if created.TenantID != sess.TenantID {
		t.Fatalf("created.TenantID = %q, want %q", created.TenantID, sess.TenantID)
	}

	// Create message
	msg := session.Message{
		ID:        "msg-pg-test-1",
		SessionID: sess.ID,
		Role:      session.RoleUser,
		Content:   "hello postgres",
		CreatedAt: now,
	}
	createdMsg, err := store.CreateMessage(ctx, msg)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if createdMsg.ID != msg.ID {
		t.Fatalf("createdMsg.ID = %q, want %q", createdMsg.ID, msg.ID)
	}

	// List messages
	messages, err := store.ListMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Content != "hello postgres" {
		t.Fatalf("messages[0].Content = %q, want %q", messages[0].Content, "hello postgres")
	}

	// List messages for nonexistent session returns error
	_, err = store.ListMessages(ctx, "nonexistent-session")
	if err == nil {
		t.Fatal("ListMessages(nonexistent) error = nil, want non-nil")
	}

	// Create message for nonexistent session returns error
	_, err = store.CreateMessage(ctx, session.Message{
		ID:        "msg-pg-orphan",
		SessionID: "nonexistent-session",
		Role:      session.RoleUser,
		Content:   "orphan",
		CreatedAt: now,
	})
	if err == nil {
		t.Fatal("CreateMessage(orphan) error = nil, want non-nil")
	}
}
