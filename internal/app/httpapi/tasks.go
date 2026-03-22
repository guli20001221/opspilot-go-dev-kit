package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
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
	TaskID           string `json:"task_id"`
	RequestID        string `json:"request_id,omitempty"`
	TenantID         string `json:"tenant_id"`
	SessionID        string `json:"session_id,omitempty"`
	TaskType         string `json:"task_type"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	ErrorReason      string `json:"error_reason,omitempty"`
	AuditRef         string `json:"audit_ref,omitempty"`
	RequiresApproval bool   `json:"requires_approval"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func (a *appHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

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

	writeJSON(w, http.StatusCreated, newTaskResponse(task))
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

	writeJSON(w, http.StatusOK, newTaskResponse(task))
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

	writeJSON(w, http.StatusOK, newTaskResponse(task))
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

	writeJSON(w, http.StatusOK, newTaskResponse(task))
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
