package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/version"
)

// VersionStore persists durable runtime version metadata in PostgreSQL.
type VersionStore struct {
	pool *pgxpool.Pool
}

// NewVersionStore constructs the version repository.
func NewVersionStore(pool *pgxpool.Pool) *VersionStore {
	return &VersionStore{pool: pool}
}

// Save inserts or updates a durable runtime version row.
func (s *VersionStore) Save(ctx context.Context, item version.Version) (version.Version, error) {
	const query = `
INSERT INTO versions (
    id,
    runtime_version,
    provider,
    model,
    prompt_bundle,
    planner_version,
    retrieval_version,
    tool_registry_version,
    critic_version,
    workflow_version,
    notes,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    runtime_version = EXCLUDED.runtime_version,
    provider = EXCLUDED.provider,
    model = EXCLUDED.model,
    prompt_bundle = EXCLUDED.prompt_bundle,
    planner_version = EXCLUDED.planner_version,
    retrieval_version = EXCLUDED.retrieval_version,
    tool_registry_version = EXCLUDED.tool_registry_version,
    critic_version = EXCLUDED.critic_version,
    workflow_version = EXCLUDED.workflow_version,
    notes = EXCLUDED.notes,
    created_at = EXCLUDED.created_at
RETURNING
    id,
    runtime_version,
    provider,
    model,
    prompt_bundle,
    planner_version,
    retrieval_version,
    tool_registry_version,
    critic_version,
    workflow_version,
    notes,
    created_at`

	row := s.pool.QueryRow(
		ctx,
		query,
		item.ID,
		item.RuntimeVersion,
		item.Provider,
		item.Model,
		item.PromptBundle,
		item.PlannerVersion,
		item.RetrievalVersion,
		item.ToolRegistryVersion,
		item.CriticVersion,
		item.WorkflowVersion,
		item.Notes,
		item.CreatedAt,
	)

	saved, err := scanVersion(row)
	if err != nil {
		return version.Version{}, err
	}

	return saved, nil
}

// Get loads a durable version by ID.
func (s *VersionStore) Get(ctx context.Context, versionID string) (version.Version, error) {
	const query = `
SELECT
    id,
    runtime_version,
    provider,
    model,
    prompt_bundle,
    planner_version,
    retrieval_version,
    tool_registry_version,
    critic_version,
    workflow_version,
    notes,
    created_at
FROM versions
WHERE id = $1`

	item, err := scanVersion(s.pool.QueryRow(ctx, query, versionID))
	if err != nil {
		if errors.Is(err, version.ErrVersionNotFound) {
			return version.Version{}, fmt.Errorf("%w: %s", version.ErrVersionNotFound, versionID)
		}

		return version.Version{}, fmt.Errorf("select version: %w", err)
	}

	return item, nil
}

// List returns a durable runtime version page.
func (s *VersionStore) List(ctx context.Context, filter version.ListFilter) (version.ListPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	const query = `
SELECT
    id,
    runtime_version,
    provider,
    model,
    prompt_bundle,
    planner_version,
    retrieval_version,
    tool_registry_version,
    critic_version,
    workflow_version,
    notes,
    created_at
FROM versions
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit+1, offset)
	if err != nil {
		return version.ListPage{}, fmt.Errorf("query versions: %w", err)
	}
	defer rows.Close()

	items := make([]version.Version, 0, limit+1)
	for rows.Next() {
		item, err := scanVersion(rows)
		if err != nil {
			return version.ListPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return version.ListPage{}, fmt.Errorf("iterate versions: %w", err)
	}

	page := version.ListPage{Versions: items}
	if len(items) > limit {
		page.HasMore = true
		page.NextOffset = offset + limit
		page.Versions = items[:limit]
	}

	return page, nil
}

func scanVersion(row pgx.Row) (version.Version, error) {
	var item version.Version
	if err := row.Scan(
		&item.ID,
		&item.RuntimeVersion,
		&item.Provider,
		&item.Model,
		&item.PromptBundle,
		&item.PlannerVersion,
		&item.RetrievalVersion,
		&item.ToolRegistryVersion,
		&item.CriticVersion,
		&item.WorkflowVersion,
		&item.Notes,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return version.Version{}, version.ErrVersionNotFound
		}

		return version.Version{}, fmt.Errorf("scan version: %w", err)
	}

	return item, nil
}
