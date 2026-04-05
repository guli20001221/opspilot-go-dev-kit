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

// LoadScopedPolicies loads matching policy rows for the given scope, keyed by scope level.
// Filters by (tenant_id, scope_level, scope_id) to return at most one row per level.
func (s *PolicyStore) LoadScopedPolicies(ctx context.Context, scope planner.PolicyScope) (map[string]planner.TenantPolicy, error) {
	const query = `
SELECT scope_level, policy_json
FROM tool_policies
WHERE tenant_id = $1
  AND (
    (scope_level = 'org' AND scope_id = $2)
    OR (scope_level = 'tenant' AND scope_id = $1)
    OR (scope_level = 'user' AND scope_id = $3)
  )
ORDER BY CASE scope_level
    WHEN 'org' THEN 1
    WHEN 'tenant' THEN 2
    WHEN 'user' THEN 3
    ELSE 4
END`

	orgID := scope.OrgID
	if orgID == "" {
		orgID = scope.TenantID // fallback: treat tenant as org when orgID not set
	}

	rows, err := s.pool.Query(ctx, query, scope.TenantID, orgID, scope.UserID)
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
				slog.String("tenant_id", scope.TenantID),
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
	}
	// When allow_tool_use is absent from JSON, AllowToolUse stays false (zero value).
	// Only explicit "allow_tool_use": true enables tools. This prevents an empty
	// policy row from accidentally overriding a parent-level tool restriction.
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
// Cache is keyed by (tenantID:userID) for per-user isolation.
func (l *HierarchicalPolicyLoader) LoadPolicy(ctx context.Context, scope planner.PolicyScope) planner.TenantPolicy {
	cacheKey := scope.TenantID + ":" + scope.UserID

	// Check cache first
	if l.cacheTTL > 0 {
		l.mu.RLock()
		cached, hasCached := l.cache[cacheKey]
		l.mu.RUnlock()
		if hasCached && time.Now().Before(cached.expiresAt) {
			return cached.policy
		}
		// Keep stale entry reference for fail-stale fallback below
		_ = cached
	}

	// Load from database
	scoped, err := l.store.LoadScopedPolicies(ctx, scope)
	if err != nil {
		slog.Warn("failed to load tenant policies",
			slog.String("tenant_id", scope.TenantID),
			slog.String("user_id", scope.UserID),
			slog.Any("error", err),
		)
		// Fail-stale: return expired cached policy rather than permissive defaults.
		// This prevents a DB outage from silently removing all restrictions.
		l.mu.RLock()
		if stale, ok := l.cache[cacheKey]; ok {
			l.mu.RUnlock()
			slog.Info("serving stale cached policy due to DB error",
				slog.String("tenant_id", scope.TenantID),
			)
			return stale.policy
		}
		l.mu.RUnlock()
		// No cache entry at all — fail-closed: deny all tools
		return planner.TenantPolicy{Configured: true, AllowToolUse: false}
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
		l.cache[cacheKey] = cachedPolicy{
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
