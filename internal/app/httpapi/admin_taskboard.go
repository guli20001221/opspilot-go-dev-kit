package httpapi

import (
	"net/http"
	"time"
)

type adminTaskBoardResponse struct {
	Items   []adminTaskBoardItemResponse  `json:"items"`
	Page    adminTaskBoardPageResponse    `json:"page"`
	Summary adminTaskBoardSummaryResponse `json:"summary"`
}

type adminTaskBoardItemResponse struct {
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

type adminTaskBoardPageResponse struct {
	HasMore    bool `json:"has_more"`
	NextOffset *int `json:"next_offset,omitempty"`
}

type adminTaskBoardSummaryResponse struct {
	VisibleCount          int                                `json:"visible_count"`
	RequiresApprovalCount int                                `json:"requires_approval_count"`
	LatestUpdatedAt       string                             `json:"latest_updated_at,omitempty"`
	LatestFailureReason   string                             `json:"latest_failure_reason,omitempty"`
	StatusCounts          adminTaskBoardStatusCountsResponse `json:"status_counts"`
	ReasonCounts          adminTaskBoardReasonCountsResponse `json:"reason_counts"`
	TaskTypeCounts        adminTaskBoardTypeCountsResponse   `json:"task_type_counts"`
}

type adminTaskBoardStatusCountsResponse struct {
	Queued          int `json:"queued"`
	Running         int `json:"running"`
	Succeeded       int `json:"succeeded"`
	Failed          int `json:"failed"`
	WaitingApproval int `json:"waiting_approval"`
}

type adminTaskBoardReasonCountsResponse struct {
	WorkflowRequired int `json:"workflow_required"`
	ApprovalRequired int `json:"approval_required"`
}

type adminTaskBoardTypeCountsResponse struct {
	ReportGeneration      int `json:"report_generation"`
	ApprovedToolExecution int `json:"approved_tool_execution"`
}

func (a *appHandler) handleAdminTaskBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	filter, err := parseTaskListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	board, err := a.adminTaskBoard.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "task_board_failed", err.Error())
		return
	}

	resp := adminTaskBoardResponse{
		Items: make([]adminTaskBoardItemResponse, 0, len(board.Items)),
		Page: adminTaskBoardPageResponse{
			HasMore: board.Page.HasMore,
		},
		Summary: adminTaskBoardSummaryResponse{
			VisibleCount:          board.Summary.VisibleCount,
			RequiresApprovalCount: board.Summary.RequiresApprovalCount,
			LatestFailureReason:   board.Summary.LatestFailureReason,
			StatusCounts: adminTaskBoardStatusCountsResponse{
				Queued:          board.Summary.StatusCounts.Queued,
				Running:         board.Summary.StatusCounts.Running,
				Succeeded:       board.Summary.StatusCounts.Succeeded,
				Failed:          board.Summary.StatusCounts.Failed,
				WaitingApproval: board.Summary.StatusCounts.WaitingApproval,
			},
			ReasonCounts: adminTaskBoardReasonCountsResponse{
				WorkflowRequired: board.Summary.ReasonCounts.WorkflowRequired,
				ApprovalRequired: board.Summary.ReasonCounts.ApprovalRequired,
			},
			TaskTypeCounts: adminTaskBoardTypeCountsResponse{
				ReportGeneration:      board.Summary.TaskTypeCounts.ReportGeneration,
				ApprovedToolExecution: board.Summary.TaskTypeCounts.ApprovedToolExecution,
			},
		},
	}
	if board.Page.NextOffset != nil {
		resp.Page.NextOffset = board.Page.NextOffset
	}
	if board.Summary.LatestUpdatedAt != nil {
		resp.Summary.LatestUpdatedAt = board.Summary.LatestUpdatedAt.Format(time.RFC3339Nano)
	}
	for _, item := range board.Items {
		resp.Items = append(resp.Items, adminTaskBoardItemResponse{
			TaskID:           item.TaskID,
			RequestID:        item.RequestID,
			TenantID:         item.TenantID,
			SessionID:        item.SessionID,
			TaskType:         item.TaskType,
			Status:           item.Status,
			Reason:           item.Reason,
			ErrorReason:      item.ErrorReason,
			AuditRef:         item.AuditRef,
			RequiresApproval: item.RequiresApproval,
			CreatedAt:        item.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:        item.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
