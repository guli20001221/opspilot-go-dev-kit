package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opspilot-go/internal/workflow"
)

type createTaskRequest struct {
	TenantID         string `json:"tenant_id"`
	SessionID        string `json:"session_id"`
	TaskType         string `json:"task_type"`
	Reason           string `json:"reason"`
	RequiresApproval bool   `json:"requires_approval"`
}

type approveTaskRequest struct {
	ApprovedBy string `json:"approved_by"`
}

type retryTaskRequest struct {
	RetriedBy string `json:"retried_by"`
}

type taskResponse struct {
	TaskID           string               `json:"task_id"`
	RequestID        string               `json:"request_id,omitempty"`
	TenantID         string               `json:"tenant_id"`
	SessionID        string               `json:"session_id,omitempty"`
	TaskType         string               `json:"task_type"`
	Status           string               `json:"status"`
	Reason           string               `json:"reason"`
	ErrorReason      string               `json:"error_reason,omitempty"`
	AuditRef         string               `json:"audit_ref,omitempty"`
	RequiresApproval bool                 `json:"requires_approval"`
	CreatedAt        string               `json:"created_at"`
	UpdatedAt        string               `json:"updated_at"`
	AuditEvents      []auditEventResponse `json:"audit_events,omitempty"`
}

type auditEventResponse struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Actor     string `json:"actor,omitempty"`
	Detail    string `json:"detail,omitempty"`
	CreatedAt string `json:"created_at"`
}

type listTasksResponse struct {
	Tasks      []taskResponse `json:"tasks"`
	HasMore    bool           `json:"has_more"`
	NextOffset *int           `json:"next_offset,omitempty"`
}

func (a *appHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListTasks(w, r)
	case http.MethodPost:
		a.handleCreateTask(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	task, err := a.workflows.Promote(r.Context(), workflow.PromoteRequest{
		RequestID:        requestIDFromRequest(r),
		TenantID:         req.TenantID,
		SessionID:        req.SessionID,
		TaskType:         req.TaskType,
		Reason:           req.Reason,
		RequiresApproval: req.RequiresApproval,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "task_create_failed", err.Error())
		return
	}

	writeTaskResponse(w, http.StatusCreated, a.workflows, r, task)
}

func (a *appHandler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTaskListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.workflows.ListTasks(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "task_list_failed", err.Error())
		return
	}

	resp := listTasksResponse{
		Tasks:   make([]taskResponse, 0, len(page.Tasks)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, task := range page.Tasks {
		resp.Tasks = append(resp.Tasks, newTaskResponse(task))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	taskID, action, ok := parseTaskPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if action == "" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		a.handleGetTask(w, r, taskID)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	switch action {
	case "approve":
		a.handleApproveTask(w, r, taskID)
	case "retry":
		a.handleRetryTask(w, r, taskID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found")
	}
}

func (a *appHandler) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := a.workflows.GetTask(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, workflow.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task_not_found", "task not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "task_lookup_failed", err.Error())
		return
	}

	writeTaskResponse(w, http.StatusOK, a.workflows, r, task)
}

func (a *appHandler) handleApproveTask(w http.ResponseWriter, r *http.Request, taskID string) {
	var req approveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	task, err := a.workflows.ApproveTask(r.Context(), taskID, req.ApprovedBy)
	if err != nil {
		writeTaskActionError(w, err)
		return
	}

	writeTaskResponse(w, http.StatusOK, a.workflows, r, task)
}

func (a *appHandler) handleRetryTask(w http.ResponseWriter, r *http.Request, taskID string) {
	var req retryTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	task, err := a.workflows.RetryTask(r.Context(), taskID, req.RetriedBy)
	if err != nil {
		writeTaskActionError(w, err)
		return
	}

	writeTaskResponse(w, http.StatusOK, a.workflows, r, task)
}

func writeTaskActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflow.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task_not_found", "task not found")
	case errors.Is(err, workflow.ErrInvalidTaskTransition):
		writeError(w, http.StatusConflict, "invalid_task_state", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "task_update_failed", err.Error())
	}
}

func parseTaskPath(path string) (taskID string, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/api/v1/tasks/")
	if trimmed == "" {
		return "", "", false
	}

	parts := strings.Split(trimmed, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", false
		}
		return parts[0], "", true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}

func parseTaskListFilter(r *http.Request) (workflow.TaskListFilter, error) {
	filter := workflow.TaskListFilter{
		TenantID: r.URL.Query().Get("tenant_id"),
		Status:   r.URL.Query().Get("status"),
		TaskType: r.URL.Query().Get("task_type"),
		Reason:   r.URL.Query().Get("reason"),
		Limit:    20,
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return workflow.TaskListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return workflow.TaskListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	if rawRequiresApproval := r.URL.Query().Get("requires_approval"); rawRequiresApproval != "" {
		requiresApproval, err := strconv.ParseBool(rawRequiresApproval)
		if err != nil {
			return workflow.TaskListFilter{}, errors.New("requires_approval must be a boolean")
		}
		filter.RequiresApproval = &requiresApproval
	}
	if rawCreatedAfter := r.URL.Query().Get("created_after"); rawCreatedAfter != "" {
		createdAfter, err := time.Parse(time.RFC3339Nano, rawCreatedAfter)
		if err != nil {
			return workflow.TaskListFilter{}, errors.New("created_after must be an RFC3339 timestamp")
		}
		filter.CreatedAfter = &createdAfter
	}
	if rawCreatedBefore := r.URL.Query().Get("created_before"); rawCreatedBefore != "" {
		createdBefore, err := time.Parse(time.RFC3339Nano, rawCreatedBefore)
		if err != nil {
			return workflow.TaskListFilter{}, errors.New("created_before must be an RFC3339 timestamp")
		}
		filter.CreatedBefore = &createdBefore
	}

	return filter, nil
}

func newTaskResponse(task workflow.Task) taskResponse {
	return taskResponse{
		TaskID:           task.ID,
		RequestID:        task.RequestID,
		TenantID:         task.TenantID,
		SessionID:        task.SessionID,
		TaskType:         task.TaskType,
		Status:           task.Status,
		Reason:           task.Reason,
		ErrorReason:      task.ErrorReason,
		AuditRef:         task.AuditRef,
		RequiresApproval: task.RequiresApproval,
		CreatedAt:        task.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:        task.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func writeTaskResponse(w http.ResponseWriter, status int, workflows *workflow.Service, r *http.Request, task workflow.Task) {
	resp := newTaskResponse(task)
	events, err := workflows.ListTaskEvents(r.Context(), task.ID)
	if err == nil {
		resp.AuditEvents = toAuditEventResponses(events)
	}
	writeJSON(w, status, resp)
}

func toAuditEventResponses(events []workflow.AuditEvent) []auditEventResponse {
	out := make([]auditEventResponse, 0, len(events))
	for _, event := range events {
		out = append(out, auditEventResponse{
			ID:        event.ID,
			Action:    event.Action,
			Actor:     event.Actor,
			Detail:    event.Detail,
			CreatedAt: event.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	return out
}
