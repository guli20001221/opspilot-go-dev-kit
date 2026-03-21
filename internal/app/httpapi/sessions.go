package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	appchat "opspilot-go/internal/app/chat"
	"opspilot-go/internal/session"
)

type appHandler struct {
	sessions *session.Service
	chat     *appchat.Service
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
	sessionService := session.NewService()

	return &appHandler{
		sessions: sessionService,
		chat:     appchat.NewService(sessionService),
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

	result, err := a.chat.Handle(r.Context(), appchat.ChatRequestEnvelope{
		RequestID:       requestIDFromRequest(r),
		TraceID:         traceIDFromRequest(r),
		TenantID:        req.TenantID,
		UserID:          req.UserID,
		SessionID:       req.SessionID,
		Mode:            req.Mode,
		UserMessage:     req.UserMessage,
		AttachmentRefs:  req.AttachmentRefs,
		ClientRequestID: req.ClientRequestID,
	})
	if err != nil {
		if req.SessionID != "" {
			writeError(w, http.StatusNotFound, "session_not_found", "session not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "chat_handle_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for _, event := range result.Events {
		writeSSE(w, event.Name, event.Data)
		flusher.Flush()
	}
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
