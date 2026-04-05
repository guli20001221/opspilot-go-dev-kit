package planner

import "sort"

// ScopeLevel identifies the hierarchy level of a tool policy.
const (
	ScopeLevelOrg    = "org"
	ScopeLevelTenant = "tenant"
	ScopeLevelUser   = "user"
)

// MergePolicies applies hierarchical inheritance: org → tenant → user.
// Each level overrides the parent's fields when they carry non-zero values.
// The result always has Configured=true if any input policy is configured.
//
// Merge rules:
//   - AllowToolUse: child wins (most specific scope decides)
//   - AllowedTools: child replaces parent list if non-empty
//   - ForbiddenTools: union of all levels (additive, never removed by child)
//   - MaxSteps: child wins if > 0
//   - RequireApprovalForWrite: true at ANY level forces true (escalation-only)
func MergePolicies(org, tenant, user TenantPolicy) TenantPolicy {
	result := TenantPolicy{}

	// Apply in order: org (broadest) → tenant → user (most specific)
	layers := []TenantPolicy{org, tenant, user}

	for _, layer := range layers {
		if !layer.Configured {
			continue
		}
		result.Configured = true
		result.AllowToolUse = layer.AllowToolUse

		if len(layer.AllowedTools) > 0 {
			result.AllowedTools = layer.AllowedTools
		}

		// ForbiddenTools are additive — a tool forbidden at org level
		// cannot be un-forbidden by a tenant or user policy.
		if len(layer.ForbiddenTools) > 0 {
			result.ForbiddenTools = unionStrings(result.ForbiddenTools, layer.ForbiddenTools)
		}

		if layer.MaxSteps > 0 {
			result.MaxSteps = layer.MaxSteps
		}

		// Approval escalation: once required at any level, cannot be removed.
		if layer.RequireApprovalForWrite {
			result.RequireApprovalForWrite = true
		}
	}

	return result
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
