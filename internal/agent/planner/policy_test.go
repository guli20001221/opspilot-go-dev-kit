package planner

import (
	"sort"
	"testing"
)

func TestMergePoliciesChildOverridesAllowToolUse(t *testing.T) {
	org := TenantPolicy{Configured: true, AllowToolUse: true}
	tenant := TenantPolicy{Configured: true, AllowToolUse: false}
	user := TenantPolicy{} // unconfigured

	got := MergePolicies(org, tenant, user)
	if got.AllowToolUse {
		t.Fatal("AllowToolUse = true, want false (tenant overrides org)")
	}
}

func TestMergePoliciesUserOverridesTenant(t *testing.T) {
	org := TenantPolicy{}
	tenant := TenantPolicy{Configured: true, AllowToolUse: false}
	user := TenantPolicy{Configured: true, AllowToolUse: true}

	got := MergePolicies(org, tenant, user)
	if !got.AllowToolUse {
		t.Fatal("AllowToolUse = false, want true (user overrides tenant)")
	}
}

func TestMergePoliciesForbiddenToolsAreAdditive(t *testing.T) {
	org := TenantPolicy{Configured: true, ForbiddenTools: []string{"dangerous_tool"}}
	tenant := TenantPolicy{Configured: true, ForbiddenTools: []string{"risky_tool"}}
	user := TenantPolicy{} // unconfigured

	got := MergePolicies(org, tenant, user)
	sort.Strings(got.ForbiddenTools)
	if len(got.ForbiddenTools) != 2 {
		t.Fatalf("ForbiddenTools = %v, want 2 items", got.ForbiddenTools)
	}
	if got.ForbiddenTools[0] != "dangerous_tool" || got.ForbiddenTools[1] != "risky_tool" {
		t.Fatalf("ForbiddenTools = %v, want [dangerous_tool, risky_tool]", got.ForbiddenTools)
	}
}

func TestMergePoliciesForbiddenCannotBeUnforbidden(t *testing.T) {
	org := TenantPolicy{Configured: true, ForbiddenTools: []string{"blocked"}}
	tenant := TenantPolicy{Configured: true} // no forbidden list = does not clear org's
	user := TenantPolicy{}

	got := MergePolicies(org, tenant, user)
	if len(got.ForbiddenTools) != 1 || got.ForbiddenTools[0] != "blocked" {
		t.Fatalf("ForbiddenTools = %v, want [blocked] (org level cannot be undone)", got.ForbiddenTools)
	}
}

func TestMergePoliciesAllowedToolsChildReplaces(t *testing.T) {
	org := TenantPolicy{Configured: true, AllowedTools: []string{"tool_a", "tool_b"}}
	tenant := TenantPolicy{Configured: true, AllowedTools: []string{"tool_a"}} // narrows to just tool_a
	user := TenantPolicy{}

	got := MergePolicies(org, tenant, user)
	if len(got.AllowedTools) != 1 || got.AllowedTools[0] != "tool_a" {
		t.Fatalf("AllowedTools = %v, want [tool_a] (tenant replaces org)", got.AllowedTools)
	}
}

func TestMergePoliciesApprovalEscalationOnlyEscalates(t *testing.T) {
	org := TenantPolicy{Configured: true, RequireApprovalForWrite: true}
	tenant := TenantPolicy{Configured: true, RequireApprovalForWrite: false} // cannot de-escalate
	user := TenantPolicy{}

	got := MergePolicies(org, tenant, user)
	if !got.RequireApprovalForWrite {
		t.Fatal("RequireApprovalForWrite = false, want true (org escalation cannot be removed)")
	}
}

func TestMergePoliciesMaxStepsChildOverrides(t *testing.T) {
	org := TenantPolicy{Configured: true, MaxSteps: 6}
	tenant := TenantPolicy{Configured: true, MaxSteps: 3}
	user := TenantPolicy{}

	got := MergePolicies(org, tenant, user)
	if got.MaxSteps != 3 {
		t.Fatalf("MaxSteps = %d, want 3 (tenant overrides org)", got.MaxSteps)
	}
}

func TestMergePoliciesAllUnconfiguredReturnsDefault(t *testing.T) {
	got := MergePolicies(TenantPolicy{}, TenantPolicy{}, TenantPolicy{})
	if got.Configured {
		t.Fatal("Configured = true, want false (all layers unconfigured)")
	}
}

func TestMergePoliciesFullThreeLevelStack(t *testing.T) {
	org := TenantPolicy{
		Configured:              true,
		AllowToolUse:            true,
		ForbiddenTools:          []string{"global_blocked"},
		MaxSteps:                6,
		RequireApprovalForWrite: true,
	}
	tenant := TenantPolicy{
		Configured:     true,
		AllowToolUse:   true,
		AllowedTools:   []string{"ticket_search", "ticket_comment_create"},
		ForbiddenTools: []string{"tenant_blocked"},
		MaxSteps:       4,
	}
	user := TenantPolicy{
		Configured:   true,
		AllowToolUse: true,
		MaxSteps:     3,
	}

	got := MergePolicies(org, tenant, user)

	if !got.Configured {
		t.Fatal("not configured")
	}
	if !got.AllowToolUse {
		t.Fatal("AllowToolUse should be true")
	}
	if len(got.AllowedTools) != 2 {
		t.Fatalf("AllowedTools = %v, want 2 items (from tenant)", got.AllowedTools)
	}
	sort.Strings(got.ForbiddenTools)
	if len(got.ForbiddenTools) != 2 {
		t.Fatalf("ForbiddenTools = %v, want 2 items (union of org+tenant)", got.ForbiddenTools)
	}
	if got.MaxSteps != 3 {
		t.Fatalf("MaxSteps = %d, want 3 (user overrides)", got.MaxSteps)
	}
	if !got.RequireApprovalForWrite {
		t.Fatal("RequireApprovalForWrite should be true (org escalation)")
	}
}
