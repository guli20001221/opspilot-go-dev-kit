package taskboard

import (
	"context"

	"opspilot-go/internal/workflow"
)

// TaskReader defines the task list operation the admin task board consumes.
type TaskReader interface {
	ListTasks(ctx context.Context, filter workflow.TaskListFilter) (workflow.TaskListPage, error)
}

// Service builds the admin task board read model from workflow task rows.
type Service struct {
	tasks TaskReader
}

// NewService constructs the admin task board service.
func NewService(tasks TaskReader) *Service {
	if tasks == nil {
		tasks = workflow.NewService()
	}

	return &Service{tasks: tasks}
}

// List returns an operator-facing task board for the provided filter.
func (s *Service) List(ctx context.Context, filter workflow.TaskListFilter) (TaskBoard, error) {
	page, err := s.tasks.ListTasks(ctx, filter)
	if err != nil {
		return TaskBoard{}, err
	}

	items := make([]TaskItem, 0, len(page.Tasks))
	summary := Summary{
		VisibleCount: len(page.Tasks),
	}
	for _, task := range page.Tasks {
		items = append(items, TaskItem{
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
			CreatedAt:        task.CreatedAt,
			UpdatedAt:        task.UpdatedAt,
		})

		if task.RequiresApproval {
			summary.RequiresApprovalCount++
		}
		switch task.Status {
		case workflow.StatusQueued:
			summary.StatusCounts.Queued++
		case workflow.StatusRunning:
			summary.StatusCounts.Running++
		case workflow.StatusSucceeded:
			summary.StatusCounts.Succeeded++
		case workflow.StatusFailed:
			summary.StatusCounts.Failed++
			if summary.LatestFailureReason == "" {
				summary.LatestFailureReason = task.ErrorReason
			}
		case workflow.StatusWaitingApproval:
			summary.StatusCounts.WaitingApproval++
		}
		switch task.Reason {
		case workflow.PromotionReasonWorkflowRequired:
			summary.ReasonCounts.WorkflowRequired++
		case workflow.PromotionReasonApprovalRequired:
			summary.ReasonCounts.ApprovalRequired++
		}
		switch task.TaskType {
		case workflow.TaskTypeReportGeneration:
			summary.TaskTypeCounts.ReportGeneration++
		case workflow.TaskTypeApprovedToolExecution:
			summary.TaskTypeCounts.ApprovedToolExecution++
		}
		if summary.LatestUpdatedAt == nil || task.UpdatedAt.After(*summary.LatestUpdatedAt) {
			updatedAt := task.UpdatedAt
			summary.LatestUpdatedAt = &updatedAt
		}
	}

	board := TaskBoard{
		Items:   items,
		Summary: summary,
		Page: PageInfo{
			HasMore: page.HasMore,
		},
	}
	if page.HasMore {
		nextOffset := page.NextOffset
		board.Page.NextOffset = &nextOffset
	}

	return board, nil
}
