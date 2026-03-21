package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"opspilot-go/internal/session"
)

type appHandler struct {
	sessions *session.Service
}

type createSessionRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
}

type chatStreamRequest struct {
	TenantID        string   `json:"tenant_id"`
	UserID          string   `json:"user_id"`
	SessionID       string   `json:"session_id"`
	Mode            string   `json:"mode"`
	UserMessage     string   `json:"user_message"`
	AttachmentRefs  []string `json:"attachment_refs"`
	ClientRequestID string   `json:"client_request_id"`
}

type listMessagesResponse struct {
	Messages []messageResponse `json:"messages"`
}

type messageResponse struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newAppHandler() *appHandler {
	return &appHandler{
		sessions: session.NewService(),
	}
}

func (a *appHandler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/sessions", a.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", a.handleSessionMessages)
	mux.HandleFunc("/api/v1/chat/stream", a.handleChatStream)
}

func (a *appHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	created, err := a.sessions.CreateSession(r.Context(), session.CreateSessionInput{
		TenantID: req.TenantID,
		UserID:   req.UserID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_create_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{SessionID: created.ID})
}

func (a *appHandler) handleSessionMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	sessionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/"), "/messages")
	if sessionID == "" || !strings.HasSuffix(r.URL.Path, "/messages") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	messages, err := a.sessions.ListMessages(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session_not_found", "session not found")
		return
	}

	resp := listMessagesResponse{Messages: make([]messageResponse, 0, len(messages))}
	for _, msg := range messages {
		resp.Messages = append(resp.Messages, messageResponse{
			ID:      msg.ID,
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "streaming unsupported")
		return
	}

	var req chatStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		created, err := a.sessions.CreateSession(r.Context(), session.CreateSessionInput{
			TenantID: req.TenantID,
			UserID:   req.UserID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "session_create_failed", err.Error())
			return
		}
		sessionID = created.ID
	}

	if _, err := a.sessions.AppendMessage(r.Context(), session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleUser,
		Content:   req.UserMessage,
	}); err != nil {
		writeError(w, http.StatusNotFound, "session_not_found", "session not found")
		return
	}

	assistantContent := "Milestone 1 placeholder response."
	if _, err := a.sessions.AppendMessage(r.Context(), session.AppendMessageInput{
		SessionID: sessionID,
		Role:      session.RoleAssistant,
		Content:   assistantContent,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "assistant_message_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSSE(w, "meta", map[string]string{
		"request_id": requestIDFromRequest(r),
		"trace_id":   traceIDFromRequest(r),
		"session_id": sessionID,
	})
	flusher.Flush()

	writeSSE(w, "state", map[string]string{"state": "completed"})
	flusher.Flush()

	writeSSE(w, "done", map[string]string{
		"session_id": sessionID,
		"content":    assistantContent,
	})
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorResponse{
		Code:    code,
		Message: message,
	})
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	data, _ := json.Marshal(payload)
	_, _ = w.Write([]byte("event: " + event + "\n"))
	_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
}
