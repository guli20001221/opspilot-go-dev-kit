package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

// EvalRunStore persists durable eval runs in PostgreSQL.
type EvalRunStore struct {
	pool *pgxpool.Pool
}

type evalRunQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewEvalRunStore constructs the eval run repository.
func NewEvalRunStore(pool *pgxpool.Pool) *EvalRunStore {
	return &EvalRunStore{pool: pool}
}

// CreateRun inserts one durable eval run row plus its immutable membership snapshot.
func (s *EvalRunStore) CreateRun(ctx context.Context, item evalsvc.EvalRun, items ...evalsvc.EvalRunItem) (evalsvc.EvalRun, error) {
	const query = `
INSERT INTO eval_runs (
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULL, NULL
)`
	const itemQuery = `
INSERT INTO eval_run_items (
    run_id,
    eval_case_id,
    position,
    title,
    source_case_id,
    source_task_id,
    source_report_id,
    trace_id,
    version_id,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, NULLIF($9, ''), $10
)`
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(
			ctx,
			query,
			item.ID,
			item.TenantID,
			item.DatasetID,
			item.DatasetName,
			item.DatasetItemCount,
			item.Status,
			item.CreatedBy,
			item.ErrorReason,
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert eval run: %w", err)
		}
		if _, err := s.appendRunEvent(ctx, tx, evalsvc.EvalRunEvent{
			RunID:     item.ID,
			Action:    evalsvc.RunEventCreated,
			Actor:     item.CreatedBy,
			Detail:    item.Status,
			CreatedAt: item.CreatedAt,
		}); err != nil {
			return err
		}
		for i, runItem := range items {
			if _, err := tx.Exec(
				ctx,
				itemQuery,
				item.ID,
				runItem.EvalCaseID,
				i,
				runItem.Title,
				runItem.SourceCaseID,
				runItem.SourceTaskID,
				runItem.SourceReportID,
				runItem.TraceID,
				runItem.VersionID,
				item.CreatedAt,
			); err != nil {
				return fmt.Errorf("insert eval run item: %w", err)
			}
		}

		return nil
	}); err != nil {
		return evalsvc.EvalRun{}, err
	}
	return s.GetRun(ctx, item.ID)
}

// GetRun loads one durable eval run by ID.
func (s *EvalRunStore) GetRun(ctx context.Context, runID string) (evalsvc.EvalRun, error) {
	const query = `
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM eval_runs
WHERE id = $1`

	var item evalsvc.EvalRun
	var startedAt, finishedAt sql.NullTime
	if err := s.pool.QueryRow(ctx, query, runID).Scan(
		&item.ID,
		&item.TenantID,
		&item.DatasetID,
		&item.DatasetName,
		&item.DatasetItemCount,
		&item.Status,
		&item.CreatedBy,
		&item.ErrorReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalRun{}, evalsvc.ErrEvalRunNotFound
		}
		return evalsvc.EvalRun{}, fmt.Errorf("scan eval run: %w", err)
	}
	if startedAt.Valid {
		item.StartedAt = startedAt.Time
	}
	if finishedAt.Valid {
		item.FinishedAt = finishedAt.Time
	}
	summaries, err := s.loadRunResultSummaries(ctx, s.pool, []string{runID})
	if err != nil {
		return evalsvc.EvalRun{}, err
	}
	item.ResultSummary = applyRunResultSummaryTotal(item.DatasetItemCount, summaries[runID])
	return item, nil
}

// GetRunDetail returns one durable eval run and a consistent snapshot of its timeline and membership.
func (s *EvalRunStore) GetRunDetail(ctx context.Context, runID string) (evalsvc.EvalRunDetail, error) {
	var item evalsvc.EvalRun
	var events []evalsvc.EvalRunEvent
	var items []evalsvc.EvalRunItem
	var results []evalsvc.EvalRunItemResult

	if err := s.withReadTx(ctx, func(tx pgx.Tx) error {
		var err error
		item, err = s.getRun(ctx, tx, runID)
		if err != nil {
			return err
		}
		events, err = s.listRunEvents(ctx, tx, runID)
		if err != nil {
			return err
		}
		items, err = s.listRunItems(ctx, tx, runID)
		if err != nil {
			return err
		}
		results, err = s.listRunItemResults(ctx, tx, runID)
		return err
	}); err != nil {
		return evalsvc.EvalRunDetail{}, err
	}

	return evalsvc.EvalRunDetail{
		Run:         withRunResultSummary(item, results),
		Events:      events,
		Items:       items,
		ItemResults: results,
	}, nil
}

// ListRuns returns one durable eval-run page with lightweight rows.
func (s *EvalRunStore) ListRuns(ctx context.Context, filter evalsvc.RunListFilter) (evalsvc.RunListPage, error) {
	const query = `
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM eval_runs
WHERE tenant_id = $1
  AND ($2 = '' OR dataset_id = $2)
  AND ($3 = '' OR status = $3)
ORDER BY updated_at DESC, id DESC
LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(ctx, query, filter.TenantID, filter.DatasetID, filter.Status, filter.Limit+1, filter.Offset)
	if err != nil {
		return evalsvc.RunListPage{}, fmt.Errorf("list eval runs: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalRun, 0, filter.Limit+1)
	for rows.Next() {
		var item evalsvc.EvalRun
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.DatasetID,
			&item.DatasetName,
			&item.DatasetItemCount,
			&item.Status,
			&item.CreatedBy,
			&item.ErrorReason,
			&item.CreatedAt,
			&item.UpdatedAt,
			&startedAt,
			&finishedAt,
		); err != nil {
			return evalsvc.RunListPage{}, fmt.Errorf("scan eval run: %w", err)
		}
		if startedAt.Valid {
			item.StartedAt = startedAt.Time
		}
		if finishedAt.Valid {
			item.FinishedAt = finishedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return evalsvc.RunListPage{}, fmt.Errorf("iterate eval runs: %w", err)
	}
	runIDs := make([]string, 0, len(items))
	for _, item := range items {
		runIDs = append(runIDs, item.ID)
	}
	summaries, err := s.loadRunResultSummaries(ctx, s.pool, runIDs)
	if err != nil {
		return evalsvc.RunListPage{}, err
	}
	for i := range items {
		items[i].ResultSummary = applyRunResultSummaryTotal(items[i].DatasetItemCount, summaries[items[i].ID])
	}

	page := evalsvc.RunListPage{Runs: items}
	if len(items) > filter.Limit {
		page.HasMore = true
		page.NextOffset = filter.Offset + filter.Limit
		page.Runs = append([]evalsvc.EvalRun(nil), items[:filter.Limit]...)
	}
	return page, nil
}

// ListRunEvents returns the append-only lifecycle timeline for one eval run.
func (s *EvalRunStore) ListRunEvents(ctx context.Context, runID string) ([]evalsvc.EvalRunEvent, error) {
	if _, err := s.GetRun(ctx, runID); err != nil {
		return nil, err
	}

	return s.listRunEvents(ctx, s.pool, runID)
}

func (s *EvalRunStore) listRunEvents(ctx context.Context, q evalRunQuerier, runID string) ([]evalsvc.EvalRunEvent, error) {

	const query = `
SELECT
    id,
    run_id,
    action,
    actor,
    detail,
    created_at
FROM eval_run_events
WHERE run_id = $1
ORDER BY created_at, id`

	rows, err := q.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("select eval run events: %w", err)
	}
	defer rows.Close()

	var events []evalsvc.EvalRunEvent
	for rows.Next() {
		var event evalsvc.EvalRunEvent
		if err := rows.Scan(&event.ID, &event.RunID, &event.Action, &event.Actor, &event.Detail, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan eval run event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run events: %w", err)
	}

	return events, nil
}

func (s *EvalRunStore) listRunItems(ctx context.Context, q evalRunQuerier, runID string) ([]evalsvc.EvalRunItem, error) {
	const query = `
SELECT
    eval_case_id,
    title,
    source_case_id,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    trace_id,
    COALESCE(version_id, '')
FROM eval_run_items
WHERE run_id = $1
ORDER BY position ASC`

	rows, err := q.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("select eval run items: %w", err)
	}
	defer rows.Close()

	var items []evalsvc.EvalRunItem
	for rows.Next() {
		var item evalsvc.EvalRunItem
		if err := rows.Scan(
			&item.EvalCaseID,
			&item.Title,
			&item.SourceCaseID,
			&item.SourceTaskID,
			&item.SourceReportID,
			&item.TraceID,
			&item.VersionID,
		); err != nil {
			return nil, fmt.Errorf("scan eval run item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run items: %w", err)
	}

	return items, nil
}

func (s *EvalRunStore) listRunItemResults(ctx context.Context, q evalRunQuerier, runID string) ([]evalsvc.EvalRunItemResult, error) {
	const query = `
SELECT
    results.eval_case_id,
    results.status,
    results.verdict,
    results.detail,
    results.score,
    results.judge_version,
    results.judge_output,
    results.updated_at
FROM eval_run_item_results results
JOIN eval_run_items items
  ON items.run_id = results.run_id
 AND items.eval_case_id = results.eval_case_id
WHERE results.run_id = $1
ORDER BY items.position ASC`

	rows, err := q.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("select eval run item results: %w", err)
	}
	defer rows.Close()

	var results []evalsvc.EvalRunItemResult
	for rows.Next() {
		var result evalsvc.EvalRunItemResult
		if err := rows.Scan(
			&result.EvalCaseID,
			&result.Status,
			&result.Verdict,
			&result.Detail,
			&result.Score,
			&result.JudgeVersion,
			&result.JudgeOutput,
			&result.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval run item result: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run item results: %w", err)
	}

	return results, nil
}

func (s *EvalRunStore) loadRunResultSummaries(ctx context.Context, q evalRunQuerier, runIDs []string) (map[string]*evalsvc.EvalRunResultSummary, error) {
	if len(runIDs) == 0 {
		return map[string]*evalsvc.EvalRunResultSummary{}, nil
	}

	const query = `
SELECT
    run_id,
    COUNT(*)::INT AS recorded_results,
    COUNT(*) FILTER (WHERE status = $2)::INT AS succeeded_items,
    COUNT(*) FILTER (WHERE status = $3)::INT AS failed_items
FROM eval_run_item_results
WHERE run_id = ANY($1)
GROUP BY run_id`

	rows, err := q.Query(ctx, query, runIDs, evalsvc.RunItemResultSucceeded, evalsvc.RunItemResultFailed)
	if err != nil {
		return nil, fmt.Errorf("select eval run result summaries: %w", err)
	}
	defer rows.Close()

	summaries := make(map[string]*evalsvc.EvalRunResultSummary, len(runIDs))
	for rows.Next() {
		var runID string
		summary := &evalsvc.EvalRunResultSummary{}
		if err := rows.Scan(&runID, &summary.RecordedResults, &summary.SucceededItems, &summary.FailedItems); err != nil {
			return nil, fmt.Errorf("scan eval run result summary: %w", err)
		}
		summary.TotalItems = summary.RecordedResults
		summaries[runID] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run result summaries: %w", err)
	}

	return summaries, nil
}

// ClaimQueuedRuns marks queued eval runs as running and returns the claimed rows.
func (s *EvalRunStore) ClaimQueuedRuns(ctx context.Context, limit int, startedAt time.Time) ([]evalsvc.EvalRun, error) {
	const query = `
WITH claimed AS (
    SELECT id
    FROM eval_runs
    WHERE status = $1
    ORDER BY created_at ASC, id ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
),
updated AS (
    UPDATE eval_runs r
    SET status = $3,
        error_reason = '',
        updated_at = $4,
        started_at = COALESCE(r.started_at, $4)
    FROM claimed
    WHERE r.id = claimed.id
    RETURNING
        r.id,
        r.tenant_id,
        r.dataset_id,
        r.dataset_name,
        r.dataset_item_count,
        r.status,
        r.created_by,
        r.error_reason,
        r.created_at,
        r.updated_at,
        r.started_at,
        r.finished_at
),
inserted_events AS (
    INSERT INTO eval_run_events (run_id, action, actor, detail, created_at)
    SELECT id, $5, $6, status, $4
    FROM updated
)
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM updated
ORDER BY created_at ASC, id ASC`

	rows, err := s.pool.Query(ctx, query, evalsvc.RunStatusQueued, limit, evalsvc.RunStatusRunning, startedAt, evalsvc.RunEventClaimed, "worker")
	if err != nil {
		return nil, fmt.Errorf("claim eval runs: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalRun, 0, limit)
	for rows.Next() {
		item, err := scanEvalRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed eval runs: %w", err)
	}

	return items, nil
}

// MarkRunSucceeded atomically finalizes a running eval run as succeeded and appends a success event.
func (s *EvalRunStore) MarkRunSucceeded(ctx context.Context, runID string, finishedAt time.Time, results []evalsvc.EvalRunItemResult) (evalsvc.EvalRun, error) {
	return s.transitionRun(ctx, runID, evalsvc.RunStatusRunning, evalsvc.RunStatusSucceeded, "", finishedAt, evalsvc.RunEventSucceeded, "worker", evalsvc.RunStatusSucceeded, results)
}

// MarkRunFailed atomically finalizes a running eval run as failed and appends a failure event.
func (s *EvalRunStore) MarkRunFailed(ctx context.Context, runID string, reason string, finishedAt time.Time, results []evalsvc.EvalRunItemResult) (evalsvc.EvalRun, error) {
	return s.transitionRun(ctx, runID, evalsvc.RunStatusRunning, evalsvc.RunStatusFailed, reason, finishedAt, evalsvc.RunEventFailed, "worker", reason, results)
}

// UpdateRun updates one durable eval run row.
func (s *EvalRunStore) UpdateRun(ctx context.Context, item evalsvc.EvalRun) (evalsvc.EvalRun, error) {
	const query = `
UPDATE eval_runs
SET tenant_id = $2,
    dataset_id = $3,
    dataset_name = $4,
    dataset_item_count = $5,
    status = $6,
    created_by = $7,
    error_reason = $8,
    created_at = $9,
    updated_at = $10,
    started_at = $11,
    finished_at = $12
WHERE id = $1`

	commandTag, err := s.pool.Exec(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.DatasetID,
		item.DatasetName,
		item.DatasetItemCount,
		item.Status,
		item.CreatedBy,
		item.ErrorReason,
		item.CreatedAt,
		item.UpdatedAt,
		nullTime(item.StartedAt),
		nullTime(item.FinishedAt),
	)
	if err != nil {
		return evalsvc.EvalRun{}, fmt.Errorf("update eval run: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return evalsvc.EvalRun{}, evalsvc.ErrEvalRunNotFound
	}

	return s.GetRun(ctx, item.ID)
}

// RetryRun atomically re-queues one failed durable eval run.
func (s *EvalRunStore) RetryRun(ctx context.Context, runID string, updatedAt time.Time) (evalsvc.EvalRun, error) {
	return s.transitionRun(ctx, runID, evalsvc.RunStatusFailed, evalsvc.RunStatusQueued, "", updatedAt, evalsvc.RunEventRetried, "operator", evalsvc.RunStatusQueued, nil)
}

func (s *EvalRunStore) transitionRun(ctx context.Context, runID string, fromStatus string, toStatus string, errorReason string, updatedAt time.Time, action string, actor string, detail string, results []evalsvc.EvalRunItemResult) (evalsvc.EvalRun, error) {
	var updated evalsvc.EvalRun

	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		const query = `
UPDATE eval_runs
SET status = $2,
    error_reason = $3,
    updated_at = $4,
    started_at = CASE WHEN $2 = $5 THEN NULL ELSE started_at END,
    finished_at = CASE WHEN $2 = $5 THEN NULL ELSE $4 END
WHERE id = $1
  AND status = $6
RETURNING
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at`

		row := tx.QueryRow(ctx, query, runID, toStatus, errorReason, updatedAt, evalsvc.RunStatusQueued, fromStatus)
		var err error
		updated, err = scanEvalRun(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return s.resolveRunTransitionMiss(ctx, tx, runID)
			}
			return err
		}
		if err := s.replaceRunItemResults(ctx, tx, runID, results); err != nil {
			return err
		}
		_, err = s.appendRunEvent(ctx, tx, evalsvc.EvalRunEvent{
			RunID:     runID,
			Action:    action,
			Actor:     actor,
			Detail:    detail,
			CreatedAt: updatedAt,
		})
		return err
	}); err != nil {
		return evalsvc.EvalRun{}, err
	}

	return withRunResultSummary(updated, results), nil
}

func (s *EvalRunStore) replaceRunItemResults(ctx context.Context, q evalRunQuerier, runID string, results []evalsvc.EvalRunItemResult) error {
	const deleteQuery = `DELETE FROM eval_run_item_results WHERE run_id = $1`
	if _, err := q.Exec(ctx, deleteQuery, runID); err != nil {
		return fmt.Errorf("delete eval run item results: %w", err)
	}
	if len(results) == 0 {
		return nil
	}

	const insertQuery = `
INSERT INTO eval_run_item_results (
    run_id,
    eval_case_id,
    status,
    verdict,
    detail,
    score,
    judge_version,
    judge_output,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for _, result := range results {
		if _, err := q.Exec(
			ctx,
			insertQuery,
			runID,
			result.EvalCaseID,
			result.Status,
			result.Verdict,
			result.Detail,
			result.Score,
			result.JudgeVersion,
			result.JudgeOutput,
			result.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert eval run item result: %w", err)
		}
	}

	return nil
}

func (s *EvalRunStore) getRun(ctx context.Context, q evalRunQuerier, runID string) (evalsvc.EvalRun, error) {
	const query = `
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM eval_runs
WHERE id = $1`

	item, err := scanEvalRun(q.QueryRow(ctx, query, runID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalRun{}, evalsvc.ErrEvalRunNotFound
		}
		return evalsvc.EvalRun{}, fmt.Errorf("scan eval run: %w", err)
	}

	return item, nil
}

func (s *EvalRunStore) resolveRunTransitionMiss(ctx context.Context, q evalRunQuerier, runID string) error {
	const query = `SELECT 1 FROM eval_runs WHERE id = $1`

	var exists int
	if err := q.QueryRow(ctx, query, runID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.ErrEvalRunNotFound
		}
		return fmt.Errorf("lookup eval run after transition miss: %w", err)
	}

	return evalsvc.ErrInvalidEvalRunState
}

func (s *EvalRunStore) appendRunEvent(ctx context.Context, q evalRunQuerier, event evalsvc.EvalRunEvent) (evalsvc.EvalRunEvent, error) {
	const query = `
INSERT INTO eval_run_events (
    run_id,
    action,
    actor,
    detail,
    created_at
) VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    run_id,
    action,
    actor,
    detail,
    created_at`

	var created evalsvc.EvalRunEvent
	if err := q.QueryRow(ctx, query, event.RunID, event.Action, event.Actor, event.Detail, event.CreatedAt).Scan(
		&created.ID,
		&created.RunID,
		&created.Action,
		&created.Actor,
		&created.Detail,
		&created.CreatedAt,
	); err != nil {
		return evalsvc.EvalRunEvent{}, fmt.Errorf("insert eval run event: %w", err)
	}

	return created, nil
}

func (s *EvalRunStore) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin eval run tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit eval run tx: %w", err)
	}

	return nil
}

func (s *EvalRunStore) withReadTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("begin eval run read tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit eval run read tx: %w", err)
	}

	return nil
}

func scanEvalRun(scanner interface {
	Scan(dest ...any) error
}) (evalsvc.EvalRun, error) {
	var item evalsvc.EvalRun
	var startedAt, finishedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.DatasetID,
		&item.DatasetName,
		&item.DatasetItemCount,
		&item.Status,
		&item.CreatedBy,
		&item.ErrorReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return evalsvc.EvalRun{}, fmt.Errorf("scan eval run: %w", err)
	}
	if startedAt.Valid {
		item.StartedAt = startedAt.Time
	}
	if finishedAt.Valid {
		item.FinishedAt = finishedAt.Time
	}
	return item, nil
}

func withRunResultSummary(item evalsvc.EvalRun, results []evalsvc.EvalRunItemResult) evalsvc.EvalRun {
	item.ResultSummary = summarizeRunResultsForTotal(item.DatasetItemCount, results)
	return item
}

func summarizeRunResultsForTotal(totalItems int, results []evalsvc.EvalRunItemResult) *evalsvc.EvalRunResultSummary {
	if len(results) == 0 {
		return nil
	}

	if totalItems == 0 {
		totalItems = len(results)
	}
	summary := &evalsvc.EvalRunResultSummary{
		TotalItems:      totalItems,
		RecordedResults: len(results),
	}
	for _, result := range results {
		switch result.Status {
		case evalsvc.RunItemResultSucceeded:
			summary.SucceededItems++
		case evalsvc.RunItemResultFailed:
			summary.FailedItems++
		}
	}
	if summary.TotalItems > summary.RecordedResults {
		summary.MissingResults = summary.TotalItems - summary.RecordedResults
	}
	return summary
}

func applyRunResultSummaryTotal(totalItems int, summary *evalsvc.EvalRunResultSummary) *evalsvc.EvalRunResultSummary {
	if summary == nil {
		return nil
	}

	adjusted := *summary
	if totalItems <= 0 {
		totalItems = adjusted.RecordedResults
	}
	adjusted.TotalItems = totalItems
	if adjusted.TotalItems > adjusted.RecordedResults {
		adjusted.MissingResults = adjusted.TotalItems - adjusted.RecordedResults
	}
	return &adjusted
}

func derefRunResultSummary(summary *evalsvc.EvalRunResultSummary) evalsvc.EvalRunResultSummary {
	if summary == nil {
		return evalsvc.EvalRunResultSummary{}
	}
	return *summary
}
