CREATE TABLE IF NOT EXISTS tool_policies (
    id          TEXT PRIMARY KEY,
    scope_level TEXT NOT NULL CHECK (scope_level IN ('org', 'tenant', 'user')),
    scope_id    TEXT NOT NULL, -- org_id, tenant_id, or user_id depending on scope_level
    tenant_id   TEXT NOT NULL, -- always present for tenant isolation
    policy_json JSONB NOT NULL DEFAULT '{}',
    version     INT NOT NULL DEFAULT 1,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tool_policies_scope
    ON tool_policies (tenant_id, scope_level, scope_id);

CREATE INDEX IF NOT EXISTS idx_tool_policies_tenant
    ON tool_policies (tenant_id);

COMMENT ON TABLE tool_policies IS 'Hierarchical tool policies: org → tenant → user. Child scopes inherit from parent and override non-zero fields.';
COMMENT ON COLUMN tool_policies.scope_level IS 'Policy scope level: org (broadest), tenant, or user (most specific)';
COMMENT ON COLUMN tool_policies.policy_json IS 'JSON policy object with fields: allow_tool_use, allowed_tools, forbidden_tools, max_steps, require_approval_for_write';
