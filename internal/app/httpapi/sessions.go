package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	admintaskboard "opspilot-go/internal/app/admin/taskboard"
	appchat "opspilot-go/internal/app/chat"
	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
	ingestpkg "opspilot-go/internal/ingestion"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/observability/tracedetail"
	"opspilot-go/internal/report"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
	toolregistry "opspilot-go/internal/tools/registry"
	"opspilot-go/internal/version"
	"opspilot-go/internal/workflow"
	adminweb "opspilot-go/web/admin"
)

type appHandler struct {
	adminTaskBoard *admintaskboard.Service
	cases          *casesvc.Service
	evalCases      *evalsvc.Service
	evalDatasets   *evalsvc.DatasetService
	evalRuns       *evalsvc.RunService
	evalReports    *evalsvc.EvalReportService
	reports        *report.Service
	traceDetails   *tracedetail.Service
	sessions       *session.Service
	versions       *version.Service
	workflows      *workflow.Service
	chat           *appchat.Service
	ingestion      *ingestpkg.Pipeline
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

func newAppHandler(workflowService *workflow.Service, reportService *report.Service, caseService *casesvc.Service, evalCaseService *evalsvc.Service, evalDatasetService *evalsvc.DatasetService, evalRunService *evalsvc.RunService, evalReportService *evalsvc.EvalReportService, versionService *version.Service, sessionService *session.Service, searchService retrieval.Searcher, llmProvider llm.Provider, ingestionPipeline *ingestpkg.Pipeline, registry *toolregistry.Registry) *appHandler {
	if sessionService == nil {
		sessionService = session.NewService()
	}
	if workflowService == nil {
		workflowService = workflow.NewService()
	}
	if reportService == nil {
		reportService = report.NewService()
	}
	if caseService == nil {
		caseService = casesvc.NewService()
	}
	if versionService == nil {
		versionService = version.NewService()
	}
	traceDetailService := tracedetail.NewService(workflowService, reportService, caseService)
	if evalCaseService == nil {
		evalCaseService = evalsvc.NewService(caseService, traceDetailService)
	}
	if evalDatasetService == nil {
		evalDatasetService = evalsvc.NewDatasetService(evalCaseService)
	}
	if evalRunService == nil {
		evalRunService = evalsvc.NewRunService(evalDatasetService)
	}
	if evalReportService == nil {
		evalReportService = evalsvc.NewEvalReportServiceWithDependencies(nil, evalRunService)
	}

	return &appHandler{
		adminTaskBoard: admintaskboard.NewService(workflowService),
		cases:          caseService,
		evalCases:      evalCaseService,
		evalDatasets:   evalDatasetService,
		evalRuns:       evalRunService,
		evalReports:    evalReportService,
		reports:        reportService,
		traceDetails:   traceDetailService,
		sessions:       sessionService,
		versions:       versionService,
		workflows:      workflowService,
		chat:           appchat.NewServiceWithLLM(sessionService, workflowService, registry, searchService, llmProvider),
		ingestion:      ingestionPipeline,
	}
}

func (a *appHandler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/task-board", a.handleAdminTaskBoardPage)
	mux.HandleFunc("/admin/cases", a.handleAdminCasesPage)
	mux.HandleFunc("/admin/evals", a.handleAdminEvalsPage)
	mux.HandleFunc("/admin/eval-datasets", a.handleAdminEvalDatasetsPage)
	mux.HandleFunc("/admin/eval-runs", a.handleAdminEvalRunsPage)
	mux.HandleFunc("/admin/eval-reports", a.handleAdminEvalReportsPage)
	mux.HandleFunc("/admin/eval-report-compare", a.handleAdminEvalReportComparePage)
	mux.HandleFunc("/admin/reports", a.handleAdminReportsPage)
	mux.HandleFunc("/admin/report-compare", a.handleAdminReportComparePage)
	mux.HandleFunc("/admin/trace-detail", a.handleAdminTraceDetailPage)
	mux.HandleFunc("/admin/version-detail", a.handleAdminVersionDetailPage)
	mux.HandleFunc("/api/v1/sessions", a.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", a.handleSessionMessages)
	mux.HandleFunc("/api/v1/admin/task-board", a.handleAdminTaskBoard)
	mux.HandleFunc("/api/v1/cases", a.handleCases)
	mux.HandleFunc("/api/v1/cases/", a.handleCaseByID)
	mux.HandleFunc("/api/v1/eval-cases", a.handleEvalCases)
	mux.HandleFunc("/api/v1/eval-cases/", a.handleEvalCaseByID)
	mux.HandleFunc("/api/v1/eval-datasets", a.handleEvalDatasets)
	mux.HandleFunc("/api/v1/eval-datasets/", a.handleEvalDatasetByID)
	mux.HandleFunc("/api/v1/eval-runs", a.handleEvalRuns)
	mux.HandleFunc("/api/v1/eval-runs/", a.handleEvalRunByID)
	mux.HandleFunc("/api/v1/eval-reports", a.handleEvalReports)
	mux.HandleFunc("/api/v1/eval-report-compare", a.handleEvalReportCompare)
	mux.HandleFunc("/api/v1/eval-reports/", a.handleEvalReportByID)
	mux.HandleFunc("/api/v1/tasks", a.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", a.handleTaskByID)
	mux.HandleFunc("/api/v1/reports", a.handleReports)
	mux.HandleFunc("/api/v1/reports/", a.handleReportByID)
	mux.HandleFunc("/api/v1/report-compare", a.handleReportCompare)
	mux.HandleFunc("/api/v1/trace-drilldown", a.handleTraceDrilldown)
	mux.HandleFunc("/api/v1/versions", a.handleVersions)
	mux.HandleFunc("/api/v1/versions/", a.handleVersionByID)
	mux.HandleFunc("/api/v1/chat/stream", a.handleChatStream)
	mux.HandleFunc("/api/v1/documents", a.handleDocuments)
}

func (a *appHandler) handleAdminTaskBoardPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/task-board" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.TaskBoardHTML())
}

func (a *appHandler) handleAdminCasesPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/cases" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.CasesHTML())
}

func (a *appHandler) handleAdminEvalsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/evals" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.EvalsHTML())
}

func (a *appHandler) handleAdminEvalDatasetsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/eval-datasets" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.EvalDatasetsHTML())
}

func (a *appHandler) handleAdminEvalRunsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/eval-runs" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.EvalRunsHTML())
}

func (a *appHandler) handleAdminEvalReportsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/eval-reports" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.EvalReportsHTML())
}

func (a *appHandler) handleAdminEvalReportComparePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/eval-report-compare" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.EvalReportCompareHTML())
}

func (a *appHandler) handleAdminReportsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/reports" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.ReportsHTML())
}

func (a *appHandler) handleAdminReportComparePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/report-compare" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.ReportCompareHTML())
}

func (a *appHandler) handleAdminTraceDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/trace-detail" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.TraceDetailHTML())
}

func (a *appHandler) handleAdminVersionDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/version-detail" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminweb.VersionDetailHTML())
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
