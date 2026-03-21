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

type taskResponse struct {
	TaskID           string `json:"task_id"`
	RequestID        string `json:"request_id,omitempty"`
	TenantID         string `json:"tenant_id"`
	SessionID        string `json:"session_id,omitempty"`
	TaskType         string `json:"task_type"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	RequiresApproval bool   `json:"requires_approval"`
	CreatedAt        string `json:"created_at"`
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
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	if taskID == "" || strings.Contains(taskID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

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

func newTaskResponse(task workflow.Task) taskResponse {
	return taskResponse{
		TaskID:           task.ID,
		RequestID:        task.RequestID,
		TenantID:         task.TenantID,
		SessionID:        task.SessionID,
		TaskType:         task.TaskType,
		Status:           task.Status,
		Reason:           task.Reason,
		RequiresApproval: task.RequiresApproval,
		CreatedAt:        task.CreatedAt.Format(time.RFC3339Nano),
	}
}
