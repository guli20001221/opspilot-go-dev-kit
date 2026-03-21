# OpsPilot-Go 实施规格（implementation-spec.md）

- 文档状态：Draft v1.0
- 适用版本：V1
- 文档目的：作为 `plan.md` 的落地补充，给 Codex / Claude Code / 开发者提供“可直接无歧义落代码”的实施规格。
- 约束来源：`AGENTS.md`、`CLAUDE.md`、目录树与技能文档。
- 优先级：当 `plan.md` 与本文件冲突时，以本文件的模块契约、状态机、字段定义和验收标准为准。

---

## 1. 版本范围与非目标

### 1.1 V1 目标（本期必须完成）

V1 只交付一个 **可运行、可演示、可追踪、可评测** 的最小闭环，包含以下能力：

1. 同步问答主链路：
   - 用户发起聊天请求；
   - 系统完成 Planner -> Retrieval / Tool -> Critic -> SSE 返回；
   - 返回内容带引用、trace_id、必要的工具审计引用。

2. 异步任务主链路：
   - 创建任务；
   - 进入 workflow 执行；
   - 可等待审批；
   - 可查看状态、结果、失败原因、报告。

3. 核心子系统：
   - API / Auth / RBAC / Tenant Guard
   - Session / Message
   - Context Engine
   - Planner / Retrieval / Tool / Critic
   - Retrieval pipeline（导入、切分、索引、召回）
   - Workflow
   - Eval / Observability
   - 管理后台最小页（任务、报告、评测）

4. 最小工具集：
   - `kb_search`：内部知识检索（系统内建）
   - `ticket_search`：工单查询（只读）
   - `ticket_comment_create`：工单评论写入（需审批）
   - `report_export`：报告导出（异步）

5. 最小评测集：
   - 至少 30 条知识问答用例；
   - 至少 10 条“检索 + 工单工具”混合用例；
   - 至少 5 条失败案例回归用例。

### 1.2 V1 非目标（明确不做）

以下内容不在 V1 范围内，Codex / Claude Code 不应自行扩展：

1. 不做微服务拆分。
2. 不做多模型动态路由和复杂成本优化。
3. 不做复杂 BPM/流程设计器。
4. 不做开放式插件市场。
5. 不做全文搜索以外的大规模推荐系统。
6. 不做跨区域多活部署。
7. 不做复杂 BI 大屏，只做最小管理后台。
8. 不做任意 SQL / 任意 URL / 任意 shell 工具暴露给模型。
9. 不做“自动写生产系统”直连能力，所有写操作必须可审批、可审计、可禁用。
10. 不做“让模型自由编排无限步工具调用”，V1 限定步数和工具集合。

### 1.3 V1 版本边界

- 单仓库、模块化单体。
- 单主模型供应商，其他供应商只保留接口，不落完整实现。
- 单租户开发环境，多租户数据模型；生产/演示环境支持多租户逻辑隔离。
- 管理后台先服务演示与排障，不追求完整产品体验。

---

## 2. 系统主链路（Canonical Flow）

### 2.1 同步请求主链路

同步链路只处理 **短任务**，目标是 5~20 秒内返回首字节，并通过 SSE 持续输出。

#### 2.1.1 请求入口

`POST /api/v1/chat/stream`

输入：
- `tenant_id`
- `session_id`（可为空，为空则创建）
- `user_message`
- `mode=chat|task`
- `attachments[]`（可选）
- `client_request_id`（可选）

处理顺序：
1. Auth：验证用户身份。
2. RBAC / Tenant Guard：验证用户属于 `tenant_id`，并拥有 `chat:run` 权限。
3. Rate Limit：按 `tenant_id + user_id` 限流。
4. Session：创建/读取 session，写入 user message。
5. Trace：创建根 span，并生成 `request_id`、`trace_id`。
6. Context Engine：读取最近会话、摘要、用户画像、租户策略、长期记忆。
7. Planner：生成结构化执行计划。
8. Executor：按计划调用 Retrieval Agent / Tool Agent。
9. Critic：审查中间结果和最终草稿。
10. Response Composer：组装最终答案，按 SSE 返回。
11. Persist：写 assistant message、tool_calls、trace 索引。

#### 2.1.2 同步链路状态机

```text
received
  -> authenticating
  -> loading_context
  -> planning
  -> executing
      -> retrieving (optional)
      -> tool_running (optional)
      -> waiting_approval (optional; if promoted, exit sync path)
  -> critic_review
  -> streaming
  -> completed
  -> failed
```

#### 2.1.3 同步链路升级为异步的条件

命中以下任一条件时，同步链路必须“升格”为异步任务：

- 预计总执行时间 > 20 秒；
- 需要人工审批；
- 批量处理 > 1 个对象；
- 需要生成文件/报告；
- Planner 输出 `requires_workflow=true`；
- Tool policy 标注 `async_only=true`。

一旦升格：
- SSE 先返回 `task_promoted` 事件；
- 响应正文中不再继续执行耗时步骤；
- 客户端改为轮询或订阅任务状态。

### 2.2 异步工作流主链路

入口：`POST /api/v1/tasks`

典型任务类型：
- `report_generation`
- `eval_run`
- `document_ingestion`
- `approved_tool_execution`

状态机：

```text
draft
  -> queued
  -> running
  -> waiting_approval
  -> approved
  -> resumed
  -> succeeded
  -> failed
  -> cancelled
  -> timed_out
```

状态说明：
- `draft`：仅创建，还未入队。
- `queued`：已提交到 workflow engine。
- `running`：有 worker 正在执行。
- `waiting_approval`：等待人工审批，不允许继续推进。
- `approved`：审批通过但 workflow 尚未恢复。
- `resumed`：workflow 恢复执行。
- `succeeded`：执行完成，产出 report/result。
- `failed`：执行失败，有结构化错误原因。
- `cancelled`：用户或管理员取消。
- `timed_out`：超时终止。

### 2.3 组件交互图

```text
Client
  -> API Gateway
  -> Auth/RBAC
  -> Session Service
  -> Context Engine
  -> Planner
  -> Retrieval Agent / Tool Agent
  -> Critic
  -> Response Composer
  -> SSE Response

Long-running path:
API -> Workflow Service -> Temporal Workflow -> Activities -> Report / Eval / Approval -> Task Result
```

---

## 3. 模块边界与输入输出契约

本节定义“谁负责什么”和“谁不负责什么”。实现时不得跨边界堆逻辑。

### 3.1 API Gateway（`cmd/api`, `internal/app`, `internal/auth`）

职责：
- HTTP 路由、SSE 输出
- 鉴权、RBAC、租户守卫、限流
- request_id / trace_id 注入
- 参数校验、错误封装
- 调用 application service

输入：HTTP 请求
输出：HTTP JSON / SSE

不负责：
- 不负责 prompt 编排
- 不负责检索逻辑
- 不负责工具业务逻辑
- 不直接操作数据库表，需经过 service/repository

### 3.2 Session Service（`internal/session`）

职责：
- session 创建/读取/关闭
- message 持久化
- 最近 N 轮会话读取
- session summary 读写触发

输入：`session_id`, `tenant_id`, `message`
输出：`Session`, `[]Message`, `SessionSummary`

不负责：
- 不负责编排 agent
- 不负责 token budget

### 3.3 Context Engine（`internal/contextengine`）

职责：
- 聚合多源上下文
- 执行 token budget 裁剪
- 生成 Planner / Retrieval / Critic 专用上下文快照
- 生成/刷新 session summary

输入：
- `ChatRequestEnvelope`
- 最近消息
- session summary
- user profile / tenant policy
- task scratchpad
- retrieval / memory hits

输出：
- `PlannerContext`
- `RetrievalContext`
- `CriticContext`
- `ContextAssemblyLog`

不负责：
- 不直接调用模型
- 不直接写最终回答

### 3.4 Planner Agent（`internal/agent/planner`）

职责：
- 意图识别
- 判断同步/异步
- 判断是否需要检索/工具/审批
- 产出结构化执行计划

输入：`PlanInput`
输出：`ExecutionPlan`

不负责：
- 不直接做检索
- 不直接调用工具
- 不直接生成最终答案

### 3.5 Retrieval Agent（`internal/agent/retrieval`, `internal/retrieval`）

职责：
- 问题重写
- 组装检索查询
- 执行召回 / 重排
- 输出 evidence blocks
- 产出 citation candidates

输入：`RetrievalRequest`
输出：`RetrievalResult`

不负责：
- 不调用外部写工具
- 不决定最终答案

### 3.6 Tool Agent（`internal/agent/tool`, `internal/tools/*`）

职责：
- 根据计划调用注册工具
- 执行输入校验
- 检查 policy（只读/写入/审批）
- 标准化工具输出
- 生成审计记录

输入：`ToolInvocation`
输出：`ToolResult | ApprovalRequired`

不负责：
- 不拼接最终自然语言答案
- 不绕过审批直接执行写操作

### 3.7 Critic Agent（`internal/agent/critic`）

职责：
- 审查答案草稿、证据、工具结果
- 检查 groundedness / 引用完整性 / 工具一致性 / 风险
- 返回“通过/修订/升级为异步/拒绝”判定

输入：`CriticInput`
输出：`CriticVerdict`

不负责：
- 不直接查库
- 不直接做工具调用

### 3.8 Workflow Service（`internal/workflow`）

职责：
- 创建任务
- 提交 workflow
- 状态迁移
- 审批挂起与恢复
- 任务级重试 / 超时 / 取消

输入：`TaskCreateRequest`, `WorkflowCommand`
输出：`Task`, `TaskState`, `ReportRef`

不负责：
- 不直接做 HTTP 展示
- 不存储会话消息

### 3.9 Eval Service（`internal/eval`）

职责：
- 管理 datasets / cases / runs
- 触发批量评测
- 存储 judge scores
- 生成回归报告

输入：`EvalRunCreateRequest`
输出：`EvalRun`, `EvalRunResult`, `RegressionSummary`

### 3.10 Observability（`internal/observability`）

职责：
- trace/span helper
- metrics
- log correlation
- 关联 request_id, task_id, session_id, trace_id

不负责：
- 不承担业务判断

---

## 4. 四个 Agent 的标准输入输出

以下 DTO 为 V1 的 **规范接口**。代码实现可以拆文件，但字段语义不可漂移。

### 4.1 Chat 主请求 DTO

```go
type ChatRequestEnvelope struct {
    RequestID       string
    TraceID         string
    TenantID        string
    UserID          string
    SessionID       string
    Mode            string // chat | task
    UserMessage     string
    AttachmentRefs  []string
    ClientRequestID string
    RequestedAt     time.Time
}
```

### 4.2 Planner 输入输出

```go
type PlanInput struct {
    Request          ChatRequestEnvelope
    Context          PlannerContext
    AvailableTools   []ToolDescriptor
    TenantPolicy     TenantPolicy
    UserPermissions  []string
}

type ExecutionPlan struct {
    PlanID                string
    Intent                string // knowledge_qa | incident_assist | ticket_update | report_request | eval_request
    RequiresRetrieval     bool
    RequiresTool          bool
    RequiresWorkflow      bool
    RequiresApproval      bool
    MaxSteps              int
    OutputSchema          string // markdown | structured_summary | json
    Steps                 []PlanStep
    PlannerReasoningShort string // brief, auditable summary only
}

type PlanStep struct {
    StepID           string
    Kind             string // retrieve | tool | synthesize | critic | promote_workflow
    Name             string
    DependsOn        []string
    ToolName         string
    ReadOnly         bool
    NeedsApproval    bool
    TimeoutSeconds   int
    Retryable        bool
    Inputs           map[string]any
}
```

约束：
- Planner 不输出自由文本步骤；必须输出结构化步骤数组。
- `MaxSteps` V1 固定 `<= 6`。
- 任一步骤若 `NeedsApproval=true`，且当前链路为同步链路，则必须升级为任务。

### 4.3 Retrieval 输入输出

```go
type RetrievalRequest struct {
    RequestID      string
    TraceID        string
    TenantID       string
    SessionID      string
    PlanID         string
    QueryText      string
    RewrittenQuery string
    Filters        RetrievalFilters
    TopK           int
    UseRerank      bool
}

type RetrievalFilters struct {
    DocumentTags []string
    TimeFrom     *time.Time
    TimeTo       *time.Time
    SourceKinds  []string
    AccessScopes []string
}

type RetrievalResult struct {
    RequestID        string
    PlanID           string
    QueryUsed        string
    EvidenceBlocks   []EvidenceBlock
    CoverageScore    float64 // 0~1
    MissingQuestions []string
}

type EvidenceBlock struct {
    EvidenceID       string
    DocumentID       string
    DocumentVersion  int
    ChunkID          string
    SourceTitle      string
    SourceURI        string
    Snippet          string
    Score            float64
    RerankScore      float64
    PublishedAt      *time.Time
    CitationLabel    string
}
```

约束：
- `TopK` 默认 8，最大 12。
- 最终进入答案合成阶段的 evidence block 最多 8 个。
- `Snippet` 必须可直接引用；不得返回仅有 chunk id 而没有可读片段。

### 4.4 Tool 输入输出

```go
type ToolInvocation struct {
    RequestID      string
    TraceID        string
    TenantID       string
    SessionID      string
    TaskID         string
    PlanID         string
    StepID         string
    ToolName       string
    ActionClass    string // read | write | admin
    RequiresApproval bool
    Arguments      json.RawMessage
    DryRun         bool
}

type ToolResult struct {
    ToolCallID      string
    ToolName        string
    Status          string // succeeded | failed | approval_required | rejected
    OutputSummary   string
    StructuredData  json.RawMessage
    ErrorCode       string
    ErrorMessage    string
    ApprovalRef     string
    AuditRef        string
}
```

约束：
- Tool agent 必须写入 `tool_calls` 审计表。
- `StructuredData` 必须是可序列化 JSON，不允许只返回自然语言。
- `write` / `admin` 类工具默认 `RequiresApproval=true`。

### 4.5 Critic 输入输出

```go
type CriticInput struct {
    Request          ChatRequestEnvelope
    Plan             ExecutionPlan
    Retrieval        *RetrievalResult
    ToolResults      []ToolResult
    DraftAnswer      string
    ExpectedSchema   string
    TenantPolicy     TenantPolicy
}

type CriticVerdict struct {
    Verdict          string // approve | revise | promote_workflow | reject
    Groundedness     float64
    CitationCoverage float64
    ToolConsistency  float64
    RiskLevel        string // low | medium | high
    MissingItems     []string
    RevisionHints    []string
    BlockingReasons  []string
}
```

通过阈值：
- `Groundedness >= 0.70`
- `CitationCoverage >= 0.80`（若 plan 需要 retrieval）
- `ToolConsistency >= 0.80`（若用了 tool）
- `RiskLevel != high`

未达标时：
- 若可补救：`verdict=revise`
- 若需要审批或长任务：`verdict=promote_workflow`
- 若明显违规：`verdict=reject`

### 4.6 最终回答 DTO

```go
type AssistantResponse struct {
    MessageID      string
    SessionID      string
    RequestID      string
    TraceID        string
    Mode           string // sync | promoted_async
    Content        string
    Citations      []CitationRef
    UsedTools      []string
    PromotedTaskID string
}

type CitationRef struct {
    CitationLabel  string
    DocumentID     string
    ChunkID        string
    SourceURI      string
}
```

---

## 5. API / SSE / Workflow 契约

### 5.1 主要 HTTP API

#### 5.1.1 Chat / Session

`POST /api/v1/chat/stream`
- 用途：同步对话入口，SSE 返回。
- 权限：`chat:run`

`POST /api/v1/sessions`
- 用途：创建 session。
- 权限：`chat:run`

`GET /api/v1/sessions/{session_id}`
- 用途：查看 session 元数据。
- 权限：`chat:read`

`GET /api/v1/sessions/{session_id}/messages`
- 用途：读取消息列表。
- 权限：`chat:read`

#### 5.1.2 Tasks / Workflow

`POST /api/v1/tasks`
- 用途：创建异步任务。
- 权限：`task:create`

`GET /api/v1/tasks/{task_id}`
- 用途：查看任务详情和状态。
- 权限：`task:read`

`POST /api/v1/tasks/{task_id}/cancel`
- 用途：取消任务。
- 权限：`task:cancel`

`POST /api/v1/tasks/{task_id}/approve`
- 用途：审批通过。
- 权限：`approval:review`

`POST /api/v1/tasks/{task_id}/reject`
- 用途：审批拒绝。
- 权限：`approval:review`

#### 5.1.3 Documents / Retrieval

`POST /api/v1/documents`
- 用途：上传文档元数据并返回 `document_id`。
- 权限：`document:write`

`POST /api/v1/documents/{document_id}/ingest`
- 用途：触发解析、切分、向量化、入索引（异步）。
- 权限：`document:write`

`GET /api/v1/documents/{document_id}`
- 用途：查看文档及版本状态。
- 权限：`document:read`

#### 5.1.4 Eval / Reports

`POST /api/v1/evals/runs`
- 用途：创建评测任务。
- 权限：`eval:run`

`GET /api/v1/evals/runs/{run_id}`
- 用途：查看评测执行结果。
- 权限：`eval:read`

`GET /api/v1/reports/{report_id}`
- 用途：查看报告。
- 权限：`report:read`

### 5.2 SSE 事件规范

`POST /api/v1/chat/stream` 必须返回 `text/event-stream`，支持以下事件：

1. `meta`
   - 首事件；包含 `request_id`, `trace_id`, `session_id`。

2. `state`
   - 表示链路状态迁移。
   - payload：`{state:"planning"}` 等。

3. `plan`
   - 可选；仅返回压缩版 planner 计划摘要，不暴露内部敏感推理。

4. `retrieval`
   - 可选；返回 evidence 命中数量和 source titles 摘要。

5. `tool`
   - 可选；返回工具调用开始/完成摘要，不暴露敏感参数。

6. `delta`
   - 文本增量。

7. `citation`
   - 引用块信息，可在 `delta` 之后分批发送。

8. `task_promoted`
   - 当同步链路升级为任务时发送。
   - payload：`{task_id,status,reason}`

9. `done`
   - 正常结束。

10. `error`
   - 异常结束。

#### 5.2.1 SSE payload 例子

```json
{
  "event": "meta",
  "data": {
    "request_id": "req_123",
    "trace_id": "tr_123",
    "session_id": "ses_123"
  }
}
```

```json
{
  "event": "task_promoted",
  "data": {
    "task_id": "task_123",
    "status": "queued",
    "reason": "approval_required"
  }
}
```

### 5.3 异步任务状态契约

任务状态字段：`status`
允许值：
- `draft`
- `queued`
- `running`
- `waiting_approval`
- `approved`
- `resumed`
- `succeeded`
- `failed`
- `cancelled`
- `timed_out`

任务返回结构：

```json
{
  "task_id": "task_123",
  "task_type": "report_generation",
  "status": "running",
  "progress": 45,
  "current_step": "collecting_ticket_data",
  "report_id": null,
  "error_code": "",
  "error_message": "",
  "created_at": "...",
  "updated_at": "..."
}
```

---

## 6. 核心数据模型（最小 DDL 草案）

以下草案用于指导 migration 和 repository 层实现。字段名允许小调整，但语义和唯一性约束不可破坏。

### 6.1 租户与用户

```sql
CREATE TABLE tenants (
  id              TEXT PRIMARY KEY,
  name            TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'active',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id              TEXT PRIMARY KEY,
  email           TEXT NOT NULL UNIQUE,
  display_name    TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'active',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tenant_memberships (
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  user_id         TEXT NOT NULL REFERENCES users(id),
  role            TEXT NOT NULL,
  permissions     JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, user_id)
);
```

### 6.2 会话与消息

```sql
CREATE TABLE sessions (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  user_id             TEXT NOT NULL REFERENCES users(id),
  title               TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL DEFAULT 'active',
  latest_summary      TEXT NOT NULL DEFAULT '',
  latest_summary_ver  INT NOT NULL DEFAULT 0,
  last_message_at     TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_tenant_user ON sessions(tenant_id, user_id, updated_at DESC);

CREATE TABLE messages (
  id                  TEXT PRIMARY KEY,
  session_id          TEXT NOT NULL REFERENCES sessions(id),
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  role                TEXT NOT NULL, -- user | assistant | system | tool
  content             TEXT NOT NULL,
  content_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
  request_id          TEXT NOT NULL,
  trace_id            TEXT NOT NULL,
  citations_json      JSONB NOT NULL DEFAULT '[]'::jsonb,
  tool_refs_json      JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_session_created ON messages(session_id, created_at ASC);
```

### 6.3 任务与审批

```sql
CREATE TABLE tasks (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  created_by          TEXT NOT NULL REFERENCES users(id),
  session_id          TEXT REFERENCES sessions(id),
  task_type           TEXT NOT NULL,
  source_request_id   TEXT NOT NULL,
  status              TEXT NOT NULL,
  progress            INT NOT NULL DEFAULT 0,
  current_step        TEXT NOT NULL DEFAULT '',
  input_json          JSONB NOT NULL,
  result_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_code          TEXT NOT NULL DEFAULT '',
  error_message       TEXT NOT NULL DEFAULT '',
  workflow_run_id     TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at         TIMESTAMPTZ
);
CREATE INDEX idx_tasks_tenant_status ON tasks(tenant_id, status, created_at DESC);

CREATE TABLE task_events (
  id                  BIGSERIAL PRIMARY KEY,
  task_id             TEXT NOT NULL REFERENCES tasks(id),
  from_status         TEXT NOT NULL,
  to_status           TEXT NOT NULL,
  event_type          TEXT NOT NULL,
  payload_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
  actor_user_id       TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_task_events_task ON task_events(task_id, id ASC);

CREATE TABLE approvals (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  task_id             TEXT NOT NULL REFERENCES tasks(id),
  tool_call_id        TEXT NOT NULL,
  status              TEXT NOT NULL, -- pending | approved | rejected | expired | cancelled
  requested_by        TEXT NOT NULL REFERENCES users(id),
  decided_by          TEXT REFERENCES users(id),
  reason              TEXT NOT NULL DEFAULT '',
  request_snapshot    JSONB NOT NULL,
  decision_note       TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at          TIMESTAMPTZ
);
CREATE INDEX idx_approvals_tenant_status ON approvals(tenant_id, status, created_at DESC);
```

### 6.4 文档、版本与分块

```sql
CREATE TABLE documents (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  source_kind         TEXT NOT NULL, -- upload | url | sync
  title               TEXT NOT NULL,
  uri                 TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL, -- uploaded | parsing | chunked | embedded | indexed | failed
  current_version     INT NOT NULL DEFAULT 0,
  tags_json           JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by          TEXT NOT NULL REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_documents_tenant_status ON documents(tenant_id, status, updated_at DESC);

CREATE TABLE document_versions (
  id                  TEXT PRIMARY KEY,
  document_id         TEXT NOT NULL REFERENCES documents(id),
  version_no          INT NOT NULL,
  content_hash        TEXT NOT NULL,
  parse_status        TEXT NOT NULL DEFAULT 'pending',
  chunk_count         INT NOT NULL DEFAULT 0,
  metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(document_id, version_no)
);

CREATE TABLE document_chunks (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  document_id         TEXT NOT NULL REFERENCES documents(id),
  document_version_id TEXT NOT NULL REFERENCES document_versions(id),
  chunk_index         INT NOT NULL,
  content             TEXT NOT NULL,
  token_count         INT NOT NULL DEFAULT 0,
  metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  embedding           VECTOR(1536),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(document_version_id, chunk_index)
);
CREATE INDEX idx_document_chunks_doc_ver ON document_chunks(document_version_id, chunk_index ASC);
```

### 6.5 工具调用与审计

```sql
CREATE TABLE tool_calls (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  session_id          TEXT REFERENCES sessions(id),
  task_id             TEXT REFERENCES tasks(id),
  request_id          TEXT NOT NULL,
  trace_id            TEXT NOT NULL,
  tool_name           TEXT NOT NULL,
  action_class        TEXT NOT NULL, -- read | write | admin
  requires_approval   BOOLEAN NOT NULL DEFAULT false,
  dry_run             BOOLEAN NOT NULL DEFAULT false,
  status              TEXT NOT NULL, -- started | succeeded | failed | approval_required | rejected
  arguments_redacted  JSONB NOT NULL DEFAULT '{}'::jsonb,
  result_summary      TEXT NOT NULL DEFAULT '',
  result_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_code          TEXT NOT NULL DEFAULT '',
  error_message       TEXT NOT NULL DEFAULT '',
  approval_id         TEXT,
  started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at         TIMESTAMPTZ
);
CREATE INDEX idx_tool_calls_request ON tool_calls(request_id, started_at ASC);
```

### 6.6 Prompt / Eval / Reports

```sql
CREATE TABLE prompt_versions (
  id                  TEXT PRIMARY KEY,
  prompt_kind         TEXT NOT NULL, -- planner | critic | judge | answer
  version_label       TEXT NOT NULL,
  content             TEXT NOT NULL,
  metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by          TEXT NOT NULL REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(prompt_kind, version_label)
);

CREATE TABLE eval_datasets (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  name                TEXT NOT NULL,
  description         TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL DEFAULT 'active',
  created_by          TEXT NOT NULL REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE eval_cases (
  id                  TEXT PRIMARY KEY,
  dataset_id          TEXT NOT NULL REFERENCES eval_datasets(id),
  case_name           TEXT NOT NULL,
  input_json          JSONB NOT NULL,
  expected_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  labels_json         JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_trace_id     TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE eval_runs (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  dataset_id          TEXT NOT NULL REFERENCES eval_datasets(id),
  status              TEXT NOT NULL,
  config_json         JSONB NOT NULL,
  summary_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by          TEXT NOT NULL REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at         TIMESTAMPTZ
);

CREATE TABLE eval_results (
  id                  TEXT PRIMARY KEY,
  run_id              TEXT NOT NULL REFERENCES eval_runs(id),
  case_id             TEXT NOT NULL REFERENCES eval_cases(id),
  status              TEXT NOT NULL,
  output_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
  scores_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
  verdict             TEXT NOT NULL DEFAULT '',
  trace_id            TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL REFERENCES tenants(id),
  report_type         TEXT NOT NULL,
  source_task_id      TEXT REFERENCES tasks(id),
  status              TEXT NOT NULL, -- generating | ready | failed
  title               TEXT NOT NULL,
  summary             TEXT NOT NULL DEFAULT '',
  content_uri         TEXT NOT NULL DEFAULT '',
  metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by          TEXT NOT NULL REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  ready_at            TIMESTAMPTZ
);
```

---

## 7. Context Engine 设计

### 7.1 上下文层次

V1 上下文按以下顺序组装：

1. `SystemPolicyBlock`
   - 固定系统规则、租户策略、工具安全规则。
2. `UserProfileBlock`
   - 用户身份、角色、权限、偏好、租户边界。
3. `RecentTurnsBlock`
   - 最近 6 条消息（user/assistant）。
4. `SessionSummaryBlock`
   - 最近摘要版本。
5. `TaskScratchpadBlock`
   - 当前 task / plan / tool 中间结果摘要。
6. `EvidenceBlockSet`
   - 检索证据块，最多 8 条。
7. `ToolResultBlockSet`
   - 工具结构化结果摘要，最多 3 组。

### 7.2 Token Budget 策略

V1 固定使用“预算比例 + 硬上限”策略：

- system/policy：10%
- user profile：5%
- recent turns：15%
- session summary：10%
- task scratchpad：10%
- evidence：40%
- tool results：10%

硬上限：
- `RecentTurnsBlock` 最多 6 条消息；
- `EvidenceBlockSet` 最多 8 条证据；
- `ToolResultBlockSet` 最多 3 个结果；
- 超限时裁剪顺序：`tool results -> recent turns -> evidence low score -> summary refresh`。

### 7.3 Summary 生成规则

满足以下任一条件时刷新 session summary：
- 新增消息后最近 12 条消息总 token 估计 > 3000；
- 距离上次 summary 已过去 8 条消息；
- workflow 恢复到 session 时需要压缩上下文。

Summary 不直接覆盖历史消息；只写入 `sessions.latest_summary` 和版本号。

### 7.4 ContextAssemblyLog

每次组装上下文必须记录：
- `request_id`
- 选入的 block 列表
- 被裁剪 block 列表
- evidence ids
- tool result refs
- 预算占用摘要

该日志用于 trace 调试和坏案例分析。

---

## 8. Tool 权限与安全策略

### 8.1 工具分类

每个工具必须声明：
- `action_class`: `read | write | admin`
- `risk_level`: `low | medium | high`
- `requires_approval`: bool
- `async_only`: bool
- `tenant_scoped`: bool
- `idempotent`: bool

### 8.2 V1 工具策略矩阵

- `read` + `low risk`：可同步执行，无需审批。
- `read` + `medium risk`：可同步执行，但必须审计。
- `write` + 任意风险：必须审批；默认异步执行。
- `admin`：默认禁用，仅管理员环境可打开。

### 8.3 审批规则

以下条件必须审批：
- 写入外部系统；
- 修改工单状态、写评论、发送通知；
- 导出包含敏感字段的报告；
- 任何跨租户数据访问尝试（应直接拒绝）。

审批记录必须保存：
- 工具名称
- 参数脱敏快照
- 发起人
- 决策人
- 决策理由
- 决策时间

### 8.4 参数安全

禁止：
- 任意 SQL
- 任意 shell
- 任意 URL 抓取
- 任意文件系统写入
- 未注册工具名称调用

必须：
- 所有工具输入使用 JSON Schema 或 Go struct 校验
- 所有工具输出写审计
- 工具错误统一标准化
- 工具超时默认 10 秒（可覆盖，但必须显式声明）

### 8.5 数据脱敏

以下字段不得直接写入日志或 SSE：
- access token / secret
- 邮箱完整地址（如无必要）
- 工单敏感正文
- 报告原始附件路径

日志和审计表应保存脱敏后的参数快照。

---

## 9. Eval / Observability 方案

### 9.1 Trace 设计

每次请求至少有以下 span：
- `api.chat.stream`
- `context.load`
- `planner.run`
- `retrieval.search`
- `tool.invoke:<tool_name>`
- `critic.run`
- `response.compose`
- `workflow.start`（若升格异步）

关键 span attributes：
- `tenant_id`
- `user_id`
- `session_id`
- `request_id`
- `task_id`
- `plan_id`
- `tool_name`
- `model_name`
- `prompt_version`

### 9.2 Metrics 设计

必须有：
- `http_request_total`
- `http_request_duration_ms`
- `llm_call_total`
- `llm_tokens_in_total`
- `llm_tokens_out_total`
- `retrieval_hits_count`
- `tool_call_total`
- `tool_call_failure_total`
- `task_run_total`
- `task_run_duration_ms`
- `approval_wait_duration_ms`
- `eval_run_total`
- `eval_case_pass_rate`

### 9.3 日志规范

所有业务日志必须包含：
- `timestamp`
- `level`
- `request_id`
- `trace_id`
- `tenant_id`
- `session_id`（如有）
- `task_id`（如有）
- `component`
- `message`

### 9.4 评测指标

V1 至少评测以下维度：
- `correctness`
- `groundedness`
- `citation_coverage`
- `tool_consistency`
- `format_adherence`
- `safety`

每项分数建议区间 `0~1`，最终 verdict：
- `pass`
- `soft_fail`
- `hard_fail`

### 9.5 坏案例沉淀

满足以下条件时应可一键沉淀为 eval case：
- 用户标记回答错误；
- Critic 判定 `reject`；
- 工具调用失败导致答案失败；
- 回答缺引用或格式不合规。

---

## 10. 技术选型与硬约束

### 10.1 选型

- 语言：Go
- API：标准库 HTTP + 中间件，SSE
- 数据库：PostgreSQL + pgvector
- DB 访问：pgx + sqlc
- 异步：Temporal
- 缓存/协调：Redis（仅需要时使用）
- 观测：OpenTelemetry + structured logging
- 模型接入：单主供应商 SDK + 内部 adapter

### 10.2 硬约束

必须：
- 所有外部工具调用可审计
- 所有写工具必须审批
- 所有关键请求可追踪到 trace_id
- Prompt / judge / rubric 版本化
- 数据访问受 tenant_id 约束
- 接口错误有 machine-readable code

禁止：
- ORM 侵入式动态查询
- 直接在 handler 中写 prompt 逻辑
- 业务核心逻辑藏在 migration 或脚本中
- 为了“快”跳过审计/审批/trace

---

## 11. 里程碑与开发顺序（Definition of Done）

### Milestone 0：Foundation

范围：
- `cmd/api`, `cmd/worker`
- 配置加载
- 健康检查
- DB migration
- 基础日志与 trace
- Makefile / dev stack

验收标准：
- `make dev-up` 可启动 PostgreSQL / Redis / Temporal / API / Worker
- `/healthz` 与 `/readyz` 可用
- 至少 1 条 migration 可执行
- request_id / trace_id 可进入日志

### Milestone 1：Session + Chat Skeleton

范围：
- sessions/messages CRUD
- `POST /api/v1/chat/stream` 打通空回答骨架
- SSE 事件 `meta/state/done/error`

验收标准：
- 能创建 session 并写入 user/assistant message
- SSE 至少稳定输出 `meta -> state -> done`
- 失败时输出统一错误码

### Milestone 2：Retrieval MVP

范围：
- documents/document_versions/document_chunks
- ingest workflow
- embedding & retrieval
- citations

验收标准：
- 可导入 1 份文档并完成分块和索引
- chat 请求可命中 evidence 并返回 citation
- 至少 10 条知识问答 case 通过率 >= 70%

### Milestone 3：Planner + Tool + Critic

范围：
- planner structured plan
- ticket_search 只读工具
- critic 基础判定
- sync main chain 完整闭环

验收标准：
- Planner 输出 `ExecutionPlan`
- Tool 调用写入 `tool_calls`
- Critic 能返回 `approve/revise/reject`
- 至少 5 条“知识 + 工单” case 可跑通

### Milestone 4：Workflow + Approval

范围：
- tasks / task_events / approvals
- Temporal workflow
- async promotion
- ticket_comment_create 审批流

验收标准：
- 同步链路遇审批可正确升格为 task
- 任务状态可从 `queued` -> `waiting_approval` -> `approved` -> `resumed` -> `succeeded`
- 审批记录完整可查

### Milestone 5：Eval + Reports + Admin MVP

范围：
- eval_datasets / eval_cases / eval_runs / eval_results
- 报告生成
- 管理后台最小页

验收标准：
- 可发起评测任务并看到 run summary
- 可查看报告与 trace 跳转
- 至少 30 + 10 + 5 套 case 可跑回归

---

## 12. 测试策略

### 12.1 单元测试

覆盖：
- planner 计划解析与约束
- context engine 裁剪逻辑
- retrieval 排序/过滤
- tool policy 判定
- critic 阈值判断

要求：
- 纯函数和 policy 逻辑必须有单测

### 12.2 仓储 / 集成测试

覆盖：
- migrations
- sqlc queries
- session/message repo
- documents/chunks repo
- tasks/approvals repo

要求：
- 使用真实 PostgreSQL 容器，不用 mock DB 替代全部仓储测试

### 12.3 API 合约测试

覆盖：
- chat stream endpoint
- tasks status endpoint
- approve/reject endpoint
- document ingest endpoint

要求：
- 验证状态码、错误码、字段、SSE 事件顺序

### 12.4 Workflow 测试

覆盖：
- 正常成功路径
- 审批挂起/恢复
- activity 重试
- 超时/取消

### 12.5 E2E 测试

最少场景：
1. 知识问答 -> retrieval -> citation -> SSE 完成
2. 工单查询 -> tool -> critic -> SSE 完成
3. 写工单评论 -> 任务创建 -> 审批 -> 恢复 -> 成功
4. 发起 eval run -> 生成报告 -> 后台可查看

### 12.6 评测回归

- 每次修改 planner/critic/prompt/tool policy 后必须至少跑关键 regression dataset。
- 若 pass rate 下降超过 5%，默认视为回归，需说明原因。

---

## 13. 代码落地建议（目录到实现映射）

- `internal/agent/planner`: `service.go`, `types.go`, `prompt.go`
- `internal/agent/retrieval`: `service.go`, `types.go`
- `internal/agent/tool`: `service.go`, `policy.go`, `types.go`
- `internal/agent/critic`: `service.go`, `types.go`
- `internal/contextengine`: `assemble.go`, `budget.go`, `summary.go`, `types.go`
- `internal/workflow`: `service.go`, `workflows.go`, `activities.go`, `types.go`
- `internal/retrieval`: `ingest.go`, `search.go`, `rerank.go`, `repository.go`
- `internal/session`: `service.go`, `repository.go`
- `internal/eval`: `service.go`, `runner.go`, `judge.go`, `repository.go`
- `pkg/apierror`: 统一错误码

---

## 14. 最终验收标准（全局）

V1 只有同时满足以下条件才算完成：

1. 至少 1 条同步链路稳定可用；
2. 至少 1 条异步审批链路稳定可用；
3. 所有工具调用均有审计记录；
4. 所有关键请求均可关联到 trace_id；
5. 至少 45 条评测用例可跑，且关键 case pass rate 达到目标；
6. 至少 1 份报告可生成并查看；
7. 管理后台可查看任务、审批、评测结果；
8. README / architecture / runbook 至少有最小版本；
9. 无未说明的核心 TODO 或“后面补”的关键安全缺口。

---

## 15. 对 Codex / Claude Code 的执行指令

1. 先按 Milestone 顺序推进，不要跨阶段大面积铺摊子。
2. 每做一个里程碑，先补 types / contract / migration，再补 service，再补 handler / workflow。
3. 任一模块开工前，先对照对应 skill。
4. 若发现 `plan.md` 与本文件冲突，优先遵守本文件的接口、状态机和字段定义。
5. 若必须偏离本规格，必须在 PR / 变更说明里写明原因与替代契约。

