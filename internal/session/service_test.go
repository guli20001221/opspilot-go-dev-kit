package session

import (
	"context"
	"testing"
)

func TestCreateSessionAndAppendMessages(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateSession(context.Background(), CreateSessionInput{
		TenantID: "tenant-1",
		UserID:   "user-1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateSession() returned empty session ID")
	}

	if _, err := svc.AppendMessage(context.Background(), AppendMessageInput{
		SessionID: created.ID,
		Role:      RoleUser,
		Content:   "hello",
	}); err != nil {
		t.Fatalf("AppendMessage(user) error = %v", err)
	}

	if _, err := svc.AppendMessage(context.Background(), AppendMessageInput{
		SessionID: created.ID,
		Role:      RoleAssistant,
		Content:   "hi",
	}); err != nil {
		t.Fatalf("AppendMessage(assistant) error = %v", err)
	}

	messages, err := svc.ListMessages(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("len(ListMessages()) = %d, want %d", len(messages), 2)
	}
	if messages[0].Role != RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", messages[0].Role, RoleUser)
	}
	if messages[1].Role != RoleAssistant {
		t.Fatalf("messages[1].Role = %q, want %q", messages[1].Role, RoleAssistant)
	}
}

func TestListMessagesRejectsUnknownSession(t *testing.T) {
	svc := NewService()

	if _, err := svc.ListMessages(context.Background(), "missing"); err == nil {
		t.Fatal("ListMessages() error = nil, want non-nil")
	}
}
