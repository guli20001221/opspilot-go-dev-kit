package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"opspilot-go/internal/agent/planner"

	"github.com/jackc/pgx/v5/pgxpool"
)

// policyRow is the database representation of one tool policy.
type policyRow struct {
	ScopeLevel string
	PolicyJSON json.RawMessage
}

// PolicyStore reads hierarchical tool policies from PostgreSQL.
type PolicyStore struct {
	pool *pgxpool.Pool
}

// NewPolicyStore constructs a PostgreSQL-backed policy store.
func NewPolicyStore(pool *pgxpool.Pool) *PolicyStore {
	return &PolicyStore{pool: pool}
}

// LoadScopedPolicies loads all policy rows for a tenant, keyed by scope level.
func (s *PolicyStore) LoadScopedPolicies(ctx context.Context, tenantID string) (map[string]planner.TenantPolicy, error) {
	const query = `
SELECT scope_level, policy_json
FROM tool_policies
WHERE tenant_id = $1
ORDER BY CASE scope_level
    WHEN 'org' THEN 1
    WHEN 'tenant' THEN 2
    WHEN 'user' THEN 3
    ELSE 4
END`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query tool_policies: %w", err)
	}
	defer rows.Close()

	result := make(map[string]planner.TenantPolicy)
	for rows.Next() {
		var row policyRow
		if err := rows.Scan(&row.ScopeLevel, &row.PolicyJSON); err != nil {
			return nil, fmt.Errorf("scan tool_policy: %w", err)
		}
		policy, err := parsePolicyJSON(row.PolicyJSON)
		if err != nil {
			slog.Warn("skipping malformed policy row",
				slog.String("scope_level", row.ScopeLevel),
				slog.String("tenant_id", tenantID),
				slog.Any("error", err),
			)
			continue
		}
		result[row.ScopeLevel] = policy
	}

	return result, rows.Err()
}

func parsePolicyJSON(raw json.RawMessage) (planner.TenantPolicy, error) {
	var parsed struct {
		AllowToolUse            *bool    `json:"allow_tool_use"`
		AllowedTools            []string `json:"allowed_tools"`
		ForbiddenTools          []string `json:"forbidden_tools"`
		MaxSteps                int      `json:"max_steps"`
		RequireApprovalForWrite bool     `json:"require_approval_for_write"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return planner.TenantPolicy{}, err
	}

	policy := planner.TenantPolicy{Configured: true}
	if parsed.AllowToolUse != nil {
		policy.AllowToolUse = *parsed.AllowToolUse
	} else {
		policy.AllowToolUse = true // default: tools allowed unless explicitly disabled
	}
	policy.AllowedTools = parsed.AllowedTools
	policy.ForbiddenTools = parsed.ForbiddenTools
	policy.MaxSteps = parsed.MaxSteps
	policy.RequireApprovalForWrite = parsed.RequireApprovalForWrite

	return policy, nil
}

// --- Cached hierarchical loader ---

// HierarchicalPolicyLoader loads and merges org→tenant→user policies
// with an in-memory cache to avoid per-request database queries.
type HierarchicalPolicyLoader struct {
	store    *PolicyStore
	cacheTTL time.Duration

	mu    sync.RWMutex
	cache map[string]cachedPolicy
}

type cachedPolicy struct {
	policy    planner.TenantPolicy
	expiresAt time.Time
}

// NewHierarchicalPolicyLoader constructs a cached hierarchical policy loader.
// cacheTTL controls how long merged policies are cached per tenant (0 = no cache).
func NewHierarchicalPolicyLoader(store *PolicyStore, cacheTTL time.Duration) *HierarchicalPolicyLoader {
	return &HierarchicalPolicyLoader{
		store:    store,
		cacheTTL: cacheTTL,
		cache:    make(map[string]cachedPolicy),
	}
}

// LoadPolicy implements planner.PolicyLoader with hierarchical merge and caching.
func (l *HierarchicalPolicyLoader) LoadPolicy(ctx context.Context, tenantID string) planner.TenantPolicy {
	// Check cache first
	if l.cacheTTL > 0 {
		l.mu.RLock()
		if cached, ok := l.cache[tenantID]; ok && time.Now().Before(cached.expiresAt) {
			l.mu.RUnlock()
			return cached.policy
		}
		l.mu.RUnlock()
	}

	// Load from database
	scoped, err := l.store.LoadScopedPolicies(ctx, tenantID)
	if err != nil {
		slog.Warn("failed to load tenant policies, using permissive defaults",
			slog.String("tenant_id", tenantID),
			slog.Any("error", err),
		)
		return planner.TenantPolicy{}
	}

	// Merge: org → tenant → user
	merged := planner.MergePolicies(
		scoped[planner.ScopeLevelOrg],
		scoped[planner.ScopeLevelTenant],
		scoped[planner.ScopeLevelUser],
	)

	// Update cache
	if l.cacheTTL > 0 {
		l.mu.Lock()
		l.cache[tenantID] = cachedPolicy{
			policy:    merged,
			expiresAt: time.Now().Add(l.cacheTTL),
		}
		l.mu.Unlock()
	}

	return merged
}

// InvalidateCache removes all cached policies, forcing a reload on next access.
func (l *HierarchicalPolicyLoader) InvalidateCache() {
	l.mu.Lock()
	l.cache = make(map[string]cachedPolicy)
	l.mu.Unlock()
}
