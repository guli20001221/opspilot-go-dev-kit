# OpsPilot-Go 项目总计划（plan.md）

- 文档状态：Draft v1.0
- 项目代号：OpsPilot-Go
- 项目类型：Golang 企业级 Multi-Agent 平台 / AgentOps 平台
- 当前目标版本：V1（可运行、可演示、可评测、可答辩）
- 适用对象：项目负责人、Codex / Claude Code、后端开发、评测/运维、答辩评审
- 更新时间：2026-03-21

---

## 1. 正式背景介绍

### 1.1 行业背景

企业对 AI Agent 的需求，已经从“一个能聊天的 Demo”快速转向“一个可执行任务、可被审计、可持续迭代的应用系统”。单纯依靠 Prompt、单步 Tool Calling 或者工作流拼接，虽然可以在演示中产生较强的直观效果，但在真实企业环境中往往存在以下问题：

1. **系统边界不清晰**：Agent 逻辑与 API、会话、权限、日志、数据库耦合在一起，无法持续演进。  
2. **任务链路不可控**：复杂请求没有经过显式的规划、路由、检索、校验，结果不稳定。  
3. **上下文粗放**：把全部历史对话塞给模型，导致成本高、噪声大、错误率上升。  
4. **缺乏企业级治理**：没有审批、审计、租户隔离、评测回归、版本对比，就很难进入真实生产场景。  
5. **无法汇报与沉淀**：没有链路数据、没有实验数据、没有失败样本库，项目无法形成可复用资产。

因此，一个真正有含金量的 Agent 项目，不是“会调用大模型”，而是“围绕大模型构建完整应用系统”的能力展示。

### 1.2 项目立项背景

OpsPilot-Go 的目标，是建设一个 **面向企业知识检索、工单协同、故障辅助分析、评测回归与报告生成** 的 Golang Multi-Agent 平台。该平台既可以作为企业内部 AI 应用的执行底座，也可以作为项目简历、技术答辩与架构展示中的代表性作品。

该项目将重点体现以下能力：

- 完整工程后端能力：API、数据库、异步工作流、权限、日志、错误处理、配置治理。  
- Agent 系统能力：Planner / Retrieval / Tool / Critic 的分工与协作。  
- 上下文工程能力：分层上下文、摘要压缩、记忆写入与检索治理。  
- AgentOps 能力：可观测、可评测、可回归、可比较、可审计。  
- 企业治理能力：审批、租户隔离、权限边界、风险兜底、版本可追踪。

### 1.3 本文档定位

本计划文档是 **项目总蓝图与阶段执行依据**。它既服务于开发实施，也服务于阶段汇报、架构评审、简历包装与答辩准备。所有代码生成、目录扩展、部署与运维动作，应优先对齐本文件的项目目标、范围、主链路和阶段规划；具体接口、状态机、字段定义与实现契约，以 `AGENTS.md` 与 `implementation-spec.md` 为准，避免各写各的。

---

## 2. 项目目标与范围

### 2.1 项目目标（V1 必达）

V1 版本要完成一个 **可运行、可演示、可追踪、可评测** 的企业级 Agent 平台最小闭环，具体包括：

1. **对话与任务双模式**  
   支持用户发起普通问答与任务式请求，例如：
   - 查询企业知识、制度、SOP；
   - 基于知识和外部工具生成处理建议；
   - 创建异步分析任务，生成报告或评测结果。

2. **四类核心 Agent 能力**  
   - Planner：意图识别、任务拆解、路由决策；
   - Retrieval：问题重写、多路检索、引用组装；
   - Tool：调用 HTTP / MCP / 内部工具；
   - Critic：校验输出质量、引用完整性、结果一致性与风险。

3. **上下文工程闭环**  
   建立分层上下文引擎，而不是简单保存聊天记录；实现：
   - 最近对话短记忆；
   - 当前任务 scratchpad；
   - 用户/租户画像；
   - 长期记忆与检索结果；
   - 摘要压缩与 token budget 管理。

4. **同步与异步主链路并存**  
   - 短链路：同步问答 + SSE 流式返回；
   - 长链路：批量评测、报告生成、审批等待等通过 Workflow 运行。

5. **AgentOps 基座**  
   对 LLM 调用、Tool 调用、检索、上下文装配、Workflow 步骤做链路追踪，并沉淀到评测数据集中。

6. **企业治理能力**  
   至少完成：
   - RBAC / 租户隔离；
   - 高风险工具审批；
   - 工具调用审计；
   - Prompt / Rubric / Eval 版本化。

7. **管理后台最小版**  
   提供任务列表、评测任务、报告查看、失败案例沉淀、版本对比入口。

### 2.2 本期明确不做（V1 不做）

为了避免 Codex / Claude Code 到处铺摊子，V1 明确不做：

1. 不做多业务域大一统平台，只聚焦 **企业知识 + 工单/任务 + 评测**。  
2. 不做全功能前台产品设计，后台和 API 能支撑演示与答辩即可。  
3. 不做多模型供应商复杂路由平台，V1 先支持一个主供应商 + 预留 adapter。  
4. 不做大量复杂 MCP Server 自研，只接少量高价值工具。  
5. 不做复杂推荐系统、复杂权限继承模型、复杂 BPMN 流程编排。  
6. 不做微服务拆分，优先采用 **模块化单体**。  
7. 不做生产级高可用多活，只做 **单区高可用 + 可恢复**。  
8. 不做花哨 UI，不追求“看起来很像产品经理原型图”，优先保证后端链路正确。

### 2.3 成功定义

V1 被视为成功，必须满足以下定义：

- 至少 1 条完整同步主链路可用；
- 至少 1 条完整异步工作流可用；
- 至少 1 组工具审批闭环可演示；
- 至少 1 套评测集与回归报告可运行；
- 至少 1 套失败案例沉淀与版本对比闭环可展示；
- 至少 1 份正式架构说明、1 份部署说明、1 份运行手册、1 套简历/答辩文案可交付。

---

## 3. 系统主链路

### 3.1 同步请求主链路

#### 3.1.1 典型用户请求

> “请帮我查询公司事故处理 SOP，并结合最近两周相关工单，给出处理建议，最后以结构化摘要返回。”

#### 3.1.2 流经路径

1. **Client -> API**  
   用户通过 Web/Admin 或 API 发起请求。API 完成鉴权、租户识别、会话识别、请求限流、生成 `request_id`。

2. **API -> Session / Context Engine**  
   API 写入用户消息，加载会话最近 N 轮对话、当前未完成任务状态、用户画像、租户配置。

3. **Context Engine -> Planner**  
   将用户目标、约束、实体、时间范围、权限范围、可用工具列表输入 Planner。

4. **Planner -> Execution Plan**  
   Planner 输出结构化计划：
   - 是否需要检索；
   - 是否需要工具；
   - 哪些步骤可并行；
   - 哪些步骤需要审批；
   - 最终输出格式要求。

5. **Planner -> Retrieval Agent / Tool Agent**  
   - Retrieval Agent：问题重写、知识召回、重排、引用证据块组装；
   - Tool Agent：对工单系统或外部工具执行只读查询。

6. **Execution Results -> Critic**  
   Critic 对中间结果做检查：
   - 证据是否充分；
   - 工具与知识结果是否冲突；
   - 是否满足输出格式；
   - 是否存在高风险内容。

7. **Critic -> Response Composer**  
   如结果足够，则生成最终回答，按 SSE 流式返回；若结果不足，则要求补检索 / 补工具 / 补摘要。

8. **Response -> Persist & Observe**  
   最终答案、引用、工具调用记录、trace、评分、失败标签写入存储层与观测层。

### 3.2 异步工作流主链路

#### 3.2.1 典型异步任务

- 批量评测某一版本 Prompt / Model / Tool Policy；
- 对一批知识文档进行导入与索引构建；
- 生成日报、周报、RCA 报告；
- 需要等待人工审批的高风险工具任务。

#### 3.2.2 流经路径

1. **Client -> API 创建任务**，返回 `task_id`。  
2. **API -> Workflow**，将任务切换为 `queued` 并投入工作流。  
3. **Workflow -> Planner / Worker Steps**，拆分为若干步骤。  
4. **需要审批时**，状态切换为 `waiting_approval`，生成审批记录。  
5. **审批通过后**，继续执行工具 / 分析 / 报告生成。  
6. **任务结束后**，产出 `report_id`、trace 链接、评测结果或错误原因。  
7. **Client 通过轮询或 SSE** 查看任务状态与结果。

### 3.3 主链路 ASCII 图

```text
[Client/Web/Admin]
        |
        v
   [API Gateway]
        |
        +--> [Auth/RBAC/Tenant Guard]
        |
        +--> [Session Store]
        |
        v
 [Context Engine]
        |
        v
     [Planner]
      /    \
     /      \
    v        v
[Retrieval] [Tool Agent] --(may require approval)--> [Approval]
     \        /
      \      /
       v    v
      [Critic]
        |
   +----+----+
   |         |
   v         v
[SSE Response] [Workflow Promotion for Long Tasks]
                 |
                 v
             [Worker / Report / Eval]
```

### 3.4 主链路状态机

同步短链路状态：

`received -> planning -> retrieving/tool_running -> critic_review -> streaming -> completed | failed`

异步工作流状态：

`queued -> running -> waiting_approval -> resumed -> succeeded | failed | cancelled | timed_out`

---

## 4. 模块边界

### 4.1 模块划分原则

- 模块按照**职责边界**划分，而不是按照“是否调用 LLM”划分。  
- 模块之间通过明确的输入输出契约协作，避免直接共享临时结构。  
- 领域逻辑不依赖 HTTP、CLI、前端或具体厂商 SDK。  
- 每个模块都必须能独立测试、独立追踪、独立审计。

### 4.2 模块边界总表

| 模块 | 目录建议 | 主要职责 | 输入 | 输出 |
|---|---|---|---|---|
| API Gateway | `cmd/api`, `internal/app`, `pkg/apierror` | 接收请求、鉴权、限流、SSE、错误封装 | HTTP 请求 | JSON / SSE / task_id |
| Auth & Tenant Guard | `internal/auth` | 身份验证、角色检查、租户隔离、资源访问约束 | token、user_id、tenant_id | auth context、deny/allow |
| Session | `internal/session` | 会话创建、消息写入、会话元数据维护 | session_id、message | 会话对象、消息对象 |
| Context Engine | `internal/contextengine` | 组装多层上下文、摘要压缩、token budget | session state、task state、profile、retrieval result | context blocks |
| Planner Agent | `internal/agent/planner` | 意图识别、步骤规划、任务路由、输出 schema 约束 | user request + context summary | execution plan |
| Retrieval Agent | `internal/agent/retrieval`, `internal/retrieval` | 问题重写、召回、重排、证据块引用 | retrieval query object | ranked evidence list |
| Tool Agent | `internal/agent/tool`, `internal/tools/*` | 调用内部/外部工具、处理审批、统一工具输出 | tool request | typed tool result |
| Critic Agent | `internal/agent/critic` | 校验内容质量、引用一致性、风险控制 | plan + evidence + tool results + draft answer | critic result |
| Workflow | `internal/workflow`, `cmd/worker` | 长任务、重试、超时、审批恢复、报告生成 | task payload | workflow state / report |
| Model Adapter | `internal/model` | 屏蔽模型供应商细节，统一请求/响应结构 | prompt/tool spec | model response |
| Observability | `internal/observability` | trace、metrics、log correlation、LLM spans | execution events | telemetry |
| Eval | `internal/eval`, `eval/*` | 数据集、评测用例、实验执行、结果汇总 | eval run spec | scores / reports |
| Storage | `internal/storage`, `db/*` | DB 访问、事务、索引管理、对象存储抽象 | repository requests | typed records |
| Admin Console | `web/admin` | 任务管理、用例管理、报告查看、版本对比 | API responses | 管理界面 |

### 4.3 核心模块输入输出细化

#### 4.3.1 Planner
- 输入：
  - `UserGoal`
  - `Constraints`
  - `EntityHints`
  - `AllowedTools`
  - `ContextSummary`
- 输出：
  - `ExecutionPlan`
  - `OutputSchema`
  - `NeedApproval`
  - `ParallelizableSteps`

#### 4.3.2 Retrieval
- 输入：
  - `RetrievalQuery{tenant_scope, query_text, entities, time_range, top_k, tags}`
- 输出：
  - `EvidenceBundle{chunks[], scores[], citations[]}`

#### 4.3.3 Tool Agent
- 输入：
  - `ToolInvocation{tool_name, operation, params, risk_level, approval_mode}`
- 输出：
  - `ToolResult{status, payload, redacted_payload, source_ref, audit_id}`

#### 4.3.4 Critic
- 输入：
  - `DraftAnswer`
  - `EvidenceBundle`
  - `ToolResults`
  - `PolicyContext`
- 输出：
  - `CriticVerdict{pass/fail/revise}`
  - `Issues[]`
  - `NextAction`

#### 4.3.5 Workflow
- 输入：
  - `WorkflowRequest{task_type, input_ref, initiated_by, tenant_id}`
- 输出：
  - `WorkflowRun`
  - `Report`
  - `ApprovalRecord`

---

## 5. 关键数据模型

> 目标：达到“可以开始写 migration、repository、sqlc query”的颗粒度。

### 5.1 核心实体关系概览

```text
Tenant 1---n UserMembership n---1 User
Tenant 1---n Session 1---n Message
Tenant 1---n Document 1---n DocumentChunk
Session 1---n TaskRun 1---n TaskStep
TaskRun 1---n ToolCall
TaskRun 0..n --- 1 WorkflowRun
TaskRun 0..n --- 1 Report
TaskRun 0..n --- n EvalScore
Tenant 1---n EvalDataset 1---n EvalCase
EvalDataset 1---n EvalRun 1---n EvalScore
ToolCall 0..1 --- 1 Approval
TaskRun n---1 PromptVersion
AuditEvent references Session / TaskRun / ToolCall / Approval / UserAction
```

### 5.2 表设计（建议最小集）

#### 5.2.1 `tenants`
- 作用：租户隔离根实体。  
- 核心字段：
  - `id` UUID PK
  - `code` TEXT UNIQUE NOT NULL
  - `name` TEXT NOT NULL
  - `status` TEXT NOT NULL (`active|disabled`)
  - `settings_json` JSONB NOT NULL DEFAULT '{}'
  - `created_at` TIMESTAMPTZ NOT NULL
  - `updated_at` TIMESTAMPTZ NOT NULL
- 索引：`code` 唯一索引。

#### 5.2.2 `users`
- 作用：平台用户。  
- 核心字段：
  - `id` UUID PK
  - `external_id` TEXT UNIQUE
  - `email` TEXT UNIQUE
  - `display_name` TEXT NOT NULL
  - `status` TEXT NOT NULL (`active|disabled`)
  - `created_at`, `updated_at`

#### 5.2.3 `user_memberships`
- 作用：用户与租户的角色关系。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK -> tenants.id
  - `user_id` UUID FK -> users.id
  - `role` TEXT NOT NULL (`admin|operator|analyst|viewer`)
  - `permissions_json` JSONB NOT NULL DEFAULT '{}'
  - `created_at`
- 约束：`(tenant_id, user_id)` UNIQUE。

#### 5.2.4 `sessions`
- 作用：会话主表。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `created_by` UUID FK -> users.id
  - `title` TEXT
  - `mode` TEXT NOT NULL (`chat|task`)
  - `status` TEXT NOT NULL (`active|archived|closed`)
  - `last_message_at` TIMESTAMPTZ
  - `created_at`, `updated_at`
- 索引：`(tenant_id, created_by, last_message_at DESC)`。

#### 5.2.5 `messages`
- 作用：会话消息与系统事件消息。  
- 字段：
  - `id` UUID PK
  - `session_id` UUID FK -> sessions.id
  - `tenant_id` UUID FK
  - `role` TEXT NOT NULL (`user|assistant|system|tool|critic`)
  - `message_type` TEXT NOT NULL (`text|event|summary|tool_result`)
  - `content_json` JSONB NOT NULL
  - `token_count` INT
  - `trace_id` TEXT
  - `created_at`
- 索引：`(session_id, created_at ASC)`。

#### 5.2.6 `task_runs`
- 作用：每次用户请求或后台任务的执行记录。  
- 字段：
  - `id` UUID PK
  - `session_id` UUID NULL FK
  - `tenant_id` UUID FK
  - `initiated_by` UUID FK -> users.id
  - `request_kind` TEXT NOT NULL (`chat|analysis|eval|report|ingest`)
  - `status` TEXT NOT NULL (`queued|planning|retrieving|tool_running|critic_review|streaming|waiting_approval|succeeded|failed|cancelled|timed_out`)
  - `input_json` JSONB NOT NULL
  - `output_json` JSONB
  - `error_code` TEXT
  - `error_message` TEXT
  - `trace_id` TEXT NOT NULL
  - `workflow_run_id` UUID NULL
  - `prompt_version_id` UUID NULL
  - `started_at` TIMESTAMPTZ
  - `finished_at` TIMESTAMPTZ
  - `created_at`, `updated_at`
- 索引：`(tenant_id, status, created_at DESC)`、`trace_id`。

#### 5.2.7 `task_steps`
- 作用：任务执行的步骤级审计与观测。  
- 字段：
  - `id` UUID PK
  - `task_run_id` UUID FK -> task_runs.id
  - `step_type` TEXT NOT NULL (`context|planner|retrieval|tool|critic|compose|workflow`)
  - `step_order` INT NOT NULL
  - `status` TEXT NOT NULL (`started|completed|failed|skipped`)
  - `input_ref_json` JSONB
  - `output_ref_json` JSONB
  - `duration_ms` INT
  - `trace_span_id` TEXT
  - `created_at`
- 约束：`(task_run_id, step_order)` UNIQUE。

#### 5.2.8 `workflow_runs`
- 作用：异步工作流主表。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `workflow_type` TEXT NOT NULL (`eval_run|report_generation|document_ingest|approval_resume`)
  - `status` TEXT NOT NULL (`queued|running|waiting_approval|succeeded|failed|cancelled|timed_out`)
  - `engine_run_id` TEXT UNIQUE
  - `input_json` JSONB NOT NULL
  - `result_json` JSONB
  - `error_message` TEXT
  - `started_at`, `finished_at`, `created_at`, `updated_at`
- 索引：`(tenant_id, workflow_type, created_at DESC)`。

#### 5.2.9 `approvals`
- 作用：高风险工具调用审批。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `task_run_id` UUID FK
  - `tool_call_id` UUID NULL FK
  - `approval_type` TEXT NOT NULL (`tool_execution|bulk_export|external_write`)
  - `status` TEXT NOT NULL (`pending|approved|rejected|expired|cancelled`)
  - `requested_by` UUID FK -> users.id
  - `approved_by` UUID NULL FK -> users.id
  - `reason` TEXT
  - `decision_note` TEXT
  - `requested_at`, `decided_at`, `expires_at`
- 索引：`(tenant_id, status, requested_at DESC)`。

#### 5.2.10 `documents`
- 作用：知识文档元数据。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `source_type` TEXT NOT NULL (`upload|url|sync|manual`)
  - `source_ref` TEXT
  - `title` TEXT NOT NULL
  - `mime_type` TEXT
  - `status` TEXT NOT NULL (`uploaded|ingesting|indexed|failed|archived`)
  - `tags_json` JSONB NOT NULL DEFAULT '[]'
  - `version` INT NOT NULL DEFAULT 1
  - `checksum` TEXT
  - `created_by` UUID FK
  - `created_at`, `updated_at`
- 索引：`(tenant_id, status, created_at DESC)`。

#### 5.2.11 `document_chunks`
- 作用：文档分块与向量检索主表。  
- 字段：
  - `id` UUID PK
  - `document_id` UUID FK -> documents.id
  - `tenant_id` UUID FK
  - `chunk_index` INT NOT NULL
  - `content_text` TEXT NOT NULL
  - `content_tsv` TSVECTOR NULL
  - `embedding` VECTOR NULL
  - `token_count` INT
  - `metadata_json` JSONB NOT NULL DEFAULT '{}'
  - `source_locator` TEXT
  - `created_at`
- 约束：`(document_id, chunk_index)` UNIQUE。
- 索引：
  - `tenant_id`
  - `GIN(content_tsv)`（若启用关键字检索）
  - 向量索引（后期根据规模选 HNSW / IVFFlat）

#### 5.2.12 `tool_definitions`
- 作用：工具目录与 schema 注册。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID NULL（NULL 表示全局工具）
  - `name` TEXT NOT NULL
  - `display_name` TEXT NOT NULL
  - `transport` TEXT NOT NULL (`http|mcp|internal`)
  - `risk_level` TEXT NOT NULL (`low|medium|high|critical`)
  - `capability_type` TEXT NOT NULL (`read|write|export|admin`)
  - `input_schema_json` JSONB NOT NULL
  - `output_schema_json` JSONB NOT NULL
  - `enabled` BOOLEAN NOT NULL DEFAULT TRUE
  - `created_at`, `updated_at`
- 约束：`(tenant_id, name)` UNIQUE。

#### 5.2.13 `tool_policies`
- 作用：工具权限、审批与审计策略。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `tool_definition_id` UUID FK
  - `role` TEXT NOT NULL
  - `approval_mode` TEXT NOT NULL (`none|optional|required|dual_control`)
  - `max_calls_per_hour` INT
  - `allow_sensitive_fields` BOOLEAN NOT NULL DEFAULT FALSE
  - `policy_json` JSONB NOT NULL DEFAULT '{}'
  - `created_at`, `updated_at`
- 约束：`(tenant_id, tool_definition_id, role)` UNIQUE。

#### 5.2.14 `tool_calls`
- 作用：每次工具调用的审计记录。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `task_run_id` UUID FK
  - `tool_definition_id` UUID FK
  - `status` TEXT NOT NULL (`pending|approved|running|succeeded|failed|blocked`)
  - `request_json` JSONB NOT NULL
  - `response_json` JSONB
  - `redacted_response_json` JSONB
  - `error_message` TEXT
  - `approval_id` UUID NULL FK
  - `trace_span_id` TEXT
  - `started_at`, `finished_at`, `created_at`
- 索引：`(task_run_id, created_at)`、`(tenant_id, tool_definition_id, created_at DESC)`。

#### 5.2.15 `prompt_versions`
- 作用：Prompt / Rubric 版本化。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID NULL
  - `name` TEXT NOT NULL
  - `prompt_type` TEXT NOT NULL (`planner|retrieval_rewrite|critic|judge|report`)
  - `version` TEXT NOT NULL
  - `content_text` TEXT NOT NULL
  - `config_json` JSONB NOT NULL DEFAULT '{}'
  - `is_active` BOOLEAN NOT NULL DEFAULT FALSE
  - `created_by` UUID FK
  - `created_at`
- 约束：`(tenant_id, name, version)` UNIQUE。

#### 5.2.16 `eval_datasets`
- 作用：评测数据集。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `name` TEXT NOT NULL
  - `dataset_type` TEXT NOT NULL (`qa|tool_use|security|retrieval|approval`)
  - `description` TEXT
  - `status` TEXT NOT NULL (`draft|active|archived`)
  - `created_by` UUID FK
  - `created_at`, `updated_at`

#### 5.2.17 `eval_cases`
- 作用：单条评测用例。  
- 字段：
  - `id` UUID PK
  - `dataset_id` UUID FK -> eval_datasets.id
  - `tenant_id` UUID FK
  - `input_json` JSONB NOT NULL
  - `expected_json` JSONB NOT NULL
  - `tags_json` JSONB NOT NULL DEFAULT '[]'
  - `golden_citations_json` JSONB
  - `active` BOOLEAN NOT NULL DEFAULT TRUE
  - `created_at`, `updated_at`

#### 5.2.18 `eval_runs`
- 作用：一次评测实验。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `dataset_id` UUID FK
  - `status` TEXT NOT NULL (`queued|running|succeeded|failed|cancelled`)
  - `target_version_json` JSONB NOT NULL
  - `summary_json` JSONB
  - `started_at`, `finished_at`, `created_at`

#### 5.2.19 `eval_scores`
- 作用：评测分数明细。  
- 字段：
  - `id` UUID PK
  - `eval_run_id` UUID FK
  - `eval_case_id` UUID FK
  - `task_run_id` UUID NULL FK
  - `score_type` TEXT NOT NULL (`correctness|relevance|citation|tool_safety|format`)
  - `score_value` NUMERIC(5,2) NOT NULL
  - `judge_type` TEXT NOT NULL (`rule|llm|human`)
  - `details_json` JSONB
  - `created_at`
- 索引：`(eval_run_id, score_type)`。

#### 5.2.20 `reports`
- 作用：结构化报告输出。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `task_run_id` UUID NULL FK
  - `workflow_run_id` UUID NULL FK
  - `report_type` TEXT NOT NULL (`analysis|eval|weekly|incident`)
  - `title` TEXT NOT NULL
  - `content_json` JSONB NOT NULL
  - `storage_uri` TEXT NULL
  - `created_by` UUID FK
  - `created_at`

#### 5.2.21 `audit_events`
- 作用：安全与操作审计总表。  
- 字段：
  - `id` UUID PK
  - `tenant_id` UUID FK
  - `actor_user_id` UUID NULL FK
  - `event_type` TEXT NOT NULL
  - `resource_type` TEXT NOT NULL
  - `resource_id` TEXT NOT NULL
  - `result` TEXT NOT NULL (`success|failure|denied`)
  - `details_json` JSONB NOT NULL DEFAULT '{}'
  - `trace_id` TEXT
  - `created_at`
- 索引：`(tenant_id, event_type, created_at DESC)`。

### 5.3 关键关系与写代码注意点

1. 所有业务主表必须带 `tenant_id`，避免跨租户数据混入。  
2. `task_runs` 是主链路骨架，`task_steps` / `tool_calls` / `workflow_runs` / `reports` 都围绕它展开。  
3. `prompt_versions` 必须关联到 `task_runs` / `eval_runs`，保证版本可追溯。  
4. `document_chunks` 既服务召回，也服务引用回放，因此 `source_locator` 不可省略。  
5. `audit_events` 不是可选项，任何审批、权限拒绝、工具执行、数据导出都要写审计。

---

## 6. API / 事件契约

### 6.1 API 设计原则

- REST First；
- 长响应优先 SSE；
- 后台任务必须有 `task_id` / `workflow_id`；
- 错误统一返回标准 envelope；
- 公开接口只返回必要字段，敏感内容必须脱敏。

### 6.2 主要 Endpoint 清单

#### 6.2.1 会话与消息

##### `POST /v1/sessions`
- 作用：创建会话。  
- 请求：
```json
{
  "mode": "chat",
  "title": "事故处理协同"
}
```
- 响应：
```json
{
  "session_id": "uuid",
  "status": "active"
}
```

##### `GET /v1/sessions/{session_id}`
- 作用：获取会话元信息。

##### `POST /v1/sessions/{session_id}/messages`
- 作用：发送消息并触发同步主链路。  
- 请求：
```json
{
  "content": "请查询公司事故处理 SOP，并结合最近两周工单给出建议",
  "response_mode": "sse",
  "output_format": "structured_summary"
}
```
- 响应：
```json
{
  "task_id": "uuid",
  "stream_url": "/v1/tasks/{task_id}/events"
}
```

#### 6.2.2 SSE 事件流

##### `GET /v1/tasks/{task_id}/events`
- 作用：订阅任务执行过程和流式结果。  
- Content-Type：`text/event-stream`

#### 6.2.3 异步任务

##### `POST /v1/tasks`
- 作用：创建异步任务，例如评测、批量报告、文档导入。  
- 请求：
```json
{
  "task_type": "eval_run",
  "payload": {
    "dataset_id": "uuid",
    "target_version": {
      "prompt": "planner:v3",
      "model": "primary"
    }
  }
}
```
- 响应：
```json
{
  "task_id": "uuid",
  "status": "queued"
}
```

##### `GET /v1/tasks/{task_id}`
- 作用：查询任务状态与结果。

##### `POST /v1/tasks/{task_id}/cancel`
- 作用：取消可取消任务。

#### 6.2.4 审批

##### `GET /v1/approvals`
- 作用：获取待审批列表。

##### `POST /v1/approvals/{approval_id}/approve`
- 请求：
```json
{
  "note": "允许执行只读导出"
}
```

##### `POST /v1/approvals/{approval_id}/reject`
- 请求：
```json
{
  "note": "目标范围过大，禁止导出"
}
```

#### 6.2.5 文档与知识

##### `POST /v1/documents`
- 作用：上传/注册文档。  
- 返回：`document_id`

##### `POST /v1/documents/{document_id}/ingest`
- 作用：触发切分、embedding、索引构建。

##### `GET /v1/documents/{document_id}`
- 作用：获取文档状态。

#### 6.2.6 评测与报告

##### `POST /v1/eval/datasets`
##### `POST /v1/eval/runs`
##### `GET /v1/eval/runs/{run_id}`
##### `GET /v1/reports/{report_id}`
##### `GET /v1/reports/{report_id}/download`

### 6.3 SSE 事件模型

统一事件字段：

```json
{
  "event": "response.delta",
  "task_id": "uuid",
  "trace_id": "trace-id",
  "ts": "2026-03-21T12:00:00Z",
  "data": {}
}
```

建议至少支持以下事件：

| 事件名 | 说明 |
|---|---|
| `task.accepted` | 任务已创建 |
| `context.composed` | 上下文组装完成 |
| `planner.started` | 开始规划 |
| `planner.completed` | 规划完成 |
| `retrieval.started` | 开始检索 |
| `retrieval.completed` | 检索完成 |
| `tool.approval.required` | 工具等待审批 |
| `tool.started` | 工具开始执行 |
| `tool.completed` | 工具执行完成 |
| `critic.started` | 开始校验 |
| `critic.completed` | 校验完成 |
| `response.delta` | 文本增量 |
| `response.completed` | 最终返回完成 |
| `workflow.promoted` | 已提升为异步工作流 |
| `task.failed` | 任务失败 |
| `heartbeat` | 心跳 |

### 6.4 异步任务状态契约

状态枚举：

- `queued`
- `running`
- `waiting_approval`
- `resumed`
- `succeeded`
- `failed`
- `cancelled`
- `timed_out`
- `rolling_back`
- `rolled_back`

任务查询响应建议：

```json
{
  "task_id": "uuid",
  "task_type": "eval_run",
  "status": "running",
  "progress": 42,
  "current_step": "judge_case_42",
  "trace_id": "trace-id",
  "report_id": null,
  "error": null
}
```

### 6.5 标准错误 envelope

```json
{
  "error": {
    "code": "TOOL_APPROVAL_REQUIRED",
    "message": "The requested action requires approval.",
    "request_id": "req-123",
    "trace_id": "trace-123",
    "details": {}
  }
}
```

---

## 7. 技术选型与硬约束

### 7.1 技术选型

- **语言**：Go（模块化单体）  
- **API**：HTTP REST + SSE  
- **工作流**：Temporal  
- **数据库**：PostgreSQL + pgvector  
- **缓存/协调**：Redis（按需）  
- **对象存储**：S3 / MinIO  
- **模型接入**：通过 `internal/model` 统一 adapter，主供应商优先使用 OpenAI Responses API  
- **工具协议**：HTTP / MCP / internal tool registry  
- **观测**：OpenTelemetry + 集中 trace backend（Langfuse）  
- **评测**：数据集 + judge + 离线回归报告  
- **SQL 访问**：`pgx` + `sqlc`  
- **日志**：`slog`  
- **前端**：管理后台最小实现，可后置开发

### 7.2 明确不用什么

- 不用 ORM 作为主数据访问方式；  
- 不用微服务拆分作为默认方案；  
- 不用把 Prompt 写死在 handler 里；  
- 不用隐式全局单例到处传递；  
- 不用“所有链路都走异步”；  
- 不用“所有能力都靠一个 agent 一把梭”。

### 7.3 非功能硬约束

1. **所有请求必须可追踪**：至少到 `request_id + trace_id + task_id`。  
2. **所有高风险工具必须审批**。  
3. **所有工具调用必须审计**。  
4. **所有租户数据必须带 `tenant_id` 并强制过滤**。  
5. **所有 Prompt / Rubric / Eval 数据必须版本化**。  
6. **所有长任务必须进入 Workflow 管理**。  
7. **所有错误必须返回标准 envelope，不允许直接把底层异常丢给用户**。  
8. **所有日志不得输出原始敏感字段**。  
9. **所有新增核心能力必须补至少一条回归用例**。  
10. **任何 destructive migration、公共 API 变更、工具写操作策略变更必须审批。**

### 7.4 哪些地方必须审批

以下动作必须审批：

- 写入/修改外部业务系统；
- 导出敏感数据；
- 批量执行 side-effecting tool；
- 涉及跨系统同步、批量更新、批量删除；
- 修改 RBAC、租户权限、工具策略；
- 执行破坏性数据库变更。

### 7.5 哪些地方必须审计

以下动作必须写 `audit_events`：

- 登录、鉴权失败、权限拒绝；
- 创建/取消任务；
- 审批申请与审批决策；
- 工具执行（成功/失败/拦截）；
- 数据导入、数据导出、报告下载；
- Prompt / Policy / Tool Policy 变更；
- 评测运行与发布决策。

### 7.6 哪些地方必须可追踪

以下步骤必须有 trace / step record：

- API 请求入口；
- 上下文组装；
- Planner；
- 检索召回与重排；
- 每个工具调用；
- Critic；
- Response compose；
- Workflow step；
- Eval case 执行。

---

## 8. 里程碑 / 开发顺序

> 原则：先打地基，再做 Retrieval，再做 Runtime，再做 Workflow，再做 Eval，最后再补后台。不允许一开始就全栈铺开。

### M0：设计冻结（1~2 天）
**目标**：冻结目录、数据模型、API 主契约、主链路。  
**输出**：`plan.md`、`AGENTS.md`、skills、repo tree。  
**退出标准**：新增需求只允许进 backlog，不允许直接改核心边界。

### M1：工程骨架（3~5 天）
**要做**：
- `cmd/api`, `cmd/worker` 可启动；
- 配置加载、日志、健康检查；
- PostgreSQL / Redis / Temporal / OTel 本地联通；
- Makefile 和本地 dev stack。

**不做**：
- 不接入真实模型；
- 不做完整前端。

**退出标准**：
- `make dev-up` 可跑；
- `/healthz` 正常；
- trace 可上报。

### M2：知识与检索基础（4~6 天）
**要做**：
- 文档上传与 ingest；
- chunk + embedding + retrieval；
- 引用结构返回；
- retrieval query object。

**退出标准**：
- 单纯知识问答闭环可用；
- 引用来源可回放；
- 至少一条检索回归测试可跑。

### M3：Agent Runtime（5~7 天）
**要做**：
- Planner / Retrieval / Tool / Critic 四类 agent 最小版；
- typed execution plan；
- SSE 流式输出；
- Context Engine V1。

**退出标准**：
- 同步主链路可演示；
- 工具只读查询可跑；
- Critic 能拦住明显不合格结果。

### M4：审批与工作流（4~6 天）
**要做**：
- 高风险工具审批；
- Temporal workflow；
- 等待审批 -> 恢复执行；
- 报告生成异步化。

**退出标准**：
- 至少 1 个需要审批的任务闭环；
- 至少 1 个异步报告生成闭环。

### M5：Eval / Observability（4~6 天）
**要做**：
- traces + metrics + task steps；
- eval datasets / cases / runs；
- 自动化评分；
- 版本对比报告。

**退出标准**：
- 至少 1 套评测数据集；
- 至少 1 份回归报告；
- 至少 1 条失败案例能进入 dataset。

### M6：管理后台最小版（3~5 天）
**要做**：
- 任务列表；
- 审批列表；
- 报告查看；
- 版本对比入口；
- trace 跳转入口。

**退出标准**：
- 后台可以支撑完整演示与答辩截图。

### M7：硬化与答辩收口（3~5 天）
**要做**：
- 文档、架构图、runbook；
- 压测 / 故障演练；
- 简历文案 / 答辩文案；
- 风险清单复盘。

**退出标准**：
- 能做 3 分钟、10 分钟、30 分钟三种粒度汇报；
- 项目在本地和 staging 可稳定演示。

---

## 9. Context Engine 设计

### 9.1 设计目标

Context Engine 的目标不是“把更多内容塞给模型”，而是：

1. 只给当前任务真正需要的上下文；  
2. 让上下文具备来源、用途、预算与淘汰策略；  
3. 让上下文写入与读取都可追踪；  
4. 让上下文成为工程模块，而不是散落在 handler 里的字符串拼接。

### 9.2 上下文分层

V1 采用 6 层结构：

1. **System Policy Block**  
   系统级规则、租户级限制、输出格式、禁止行为。

2. **Recent Conversation Block**  
   最近 N 轮对话，保留语义连续性。

3. **Task Scratchpad Block**  
   当前任务的阶段性结论、已完成步骤、待执行步骤、工具结果摘要。

4. **User / Tenant Profile Block**  
   用户角色、部门、权限、偏好、租户开关。

5. **Retrieved Evidence Block**  
   检索返回的证据块、引用标识、时间范围、可信度。

6. **Memory Summary Block**  
   中长期摘要记忆，用于承接上下文压缩与长期状态。

### 9.3 组装算法

1. 解析用户输入，抽取实体、约束、时间范围；  
2. 判断是否已有未完成 task scratchpad；  
3. 读取 recent turns；  
4. 根据任务类型决定是否触发 retrieval；  
5. 把 evidence block 加入候选上下文；  
6. 按 token budget 分配上下文配额；  
7. 超预算时按顺序淘汰：
   - 低价值历史对话；
   - 过期 scratchpad；
   - 低分 evidence；
   - 可重建 summary 旧版本；
8. 生成最终 `ContextBundle`，并记录每一块被纳入的理由。

### 9.4 Token Budget 建议

V1 可采用以下经验预算：

- System Policy：10%
- Recent Conversation：20%
- Scratchpad：20%
- User/Tenant Profile：10%
- Retrieved Evidence：30%
- Memory Summary：10%

> 不是固定配比，而是默认上限。若任务明显依赖检索，可提升 Evidence 配额。

### 9.5 记忆写入策略

- 不把所有对话都写成长期记忆；  
- 只有满足以下条件的结果才写入 `Memory Summary`：
  - 用户显式要求后续沿用；
  - 对后续任务有稳定价值；
  - 不包含敏感临时细节；
  - 已过 Critic 检查。

### 9.6 关键约束

- Retrieval 输入不得直接使用全部历史对话；  
- Planner 看到的是结构化摘要，不是原始全量 transcript；  
- Tool 返回结果必须先摘要/脱敏再进入上下文；  
- Context Engine 每次输出都要记录 `context_blocks[]` 与 `include_reason`。

---

## 10. Tool 权限与安全策略

### 10.1 工具分类

| 类别 | 定义 | 示例 | 默认策略 |
|---|---|---|---|
| Read-only | 不产生外部副作用 | 查询工单、查知识、查日历 | 允许直接执行，仍需审计 |
| Advisory | 只生成建议、不落库 | 生成报告草稿 | 允许执行，输出需标记为建议 |
| Side-effecting | 会修改外部系统 | 创建工单、更新状态 | 需要审批或受控开关 |
| High-risk | 涉及敏感数据/批量操作/管理权限 | 批量导出、权限修改 | 必须审批 + 双重审计 |

### 10.2 权限控制模型

采用三层控制：

1. **身份与角色层**：用户是否具备访问能力；  
2. **工具策略层**：该角色是否能调用该工具；  
3. **审批层**：即便可调用，是否还需要审批。

### 10.3 最小权限原则

- Tool Agent 默认只暴露 allowlist 工具；  
- 不允许模型自行“发现”未注册工具；  
- 工具参数必须经过 schema 校验；  
- 所有 side-effecting tool 默认关闭，需显式启用。

### 10.4 工具执行前检查

每次工具调用前必须完成：

- 租户校验；
- 角色校验；
- policy 读取；
- 参数 schema 验证；
- 风险级别判断；
- 审批要求判断；
- trace span 创建；
- audit pending event 写入。

### 10.5 工具返回后处理

- 原始结果不得直接写日志；  
- 先生成 `redacted_response_json`；  
- 对敏感字段做脱敏；  
- 统一映射为 typed result；  
- 再注入 task scratchpad 或 response composer。

### 10.6 MCP / HTTP 工具接入策略

- 所有 MCP / HTTP 工具都必须经过统一 registry；  
- 不允许在业务代码中临时硬编码 URL 和 Token；  
- 对外调用统一带 `request_id` / `trace_id` / `tenant_id`；  
- 工具 timeout、retry、circuit breaker 在统一封装层实现。

### 10.7 安全底线

- 不允许跨租户查询；  
- 不允许用户通过 prompt 强行越权调用工具；  
- 不允许把生产凭证直接暴露给模型；  
- 不允许在无审批情况下执行高风险写操作；  
- 不允许输出完整敏感记录到前端。

---

## 11. Eval / Observability 方案

### 11.1 目标

Eval / Observability 不是附加项，而是项目核心能力。目标包括：

1. 看得见：知道每次请求做了什么；  
2. 查得到：知道问题出在哪里；  
3. 比得出：知道版本 A 和版本 B 差多少；  
4. 退得回：坏版本可以回滚；  
5. 沉淀得住：线上坏案例能进入离线评测集。

### 11.2 观测层设计

#### Trace 维度
- request trace
- task trace
- workflow trace
- tool call span
- retrieval span
- critic span

#### Metrics 维度
- 请求量 / 成功率 / 失败率
- 首 token 延迟 / 完成延迟
- 检索耗时 / 命中率
- 工具调用成功率 / 审批等待时长
- workflow 成功率 / 重试次数
- 评测平均分 / 各分项分布

#### Logs 维度
- 结构化日志
- 按 `request_id` / `task_id` / `trace_id` 关联
- 默认脱敏

### 11.3 评测层设计

#### 评测数据集类型
- `qa`：普通问答正确性
- `retrieval`：召回质量与引用完整性
- `tool_use`：工具选择与参数正确性
- `security`：越权、注入、敏感数据保护
- `approval`：审批路径是否正确

#### 评分方式
- 规则评分：格式、字段完整性、是否有引用；
- 模型评分：正确性、相关性、可读性；
- 人工评分：关键案例复核。

#### 评测输出
- case 级得分；
- run 级汇总；
- 版本对比；
- 失败榜单；
- 回归趋势。

### 11.4 坏案例沉淀机制

坏案例来源：
- 线上失败任务；
- 用户差评/人工标注；
- Critic 拦截案例；
- 工具审批被拒案例；
- 安全测试样本。

处理流程：
1. 标记 bad case；
2. 归类原因；
3. 进入 `eval_cases`；
4. 下一次版本升级前必须回归。

### 11.5 发布门禁建议

以下任一条件不满足，不允许标记为可发布版本：

- 核心数据集平均分未达到阈值；
- 安全类评测未通过；
- 批量评测失败率超阈值；
- 工具调用成功率明显下降；
- 关键链路 trace 丢失比例过高。

---

## 12. 验收标准

### 12.1 功能验收

1. 用户可以创建会话并发起请求；  
2. 同步主链路可以完成至少一种“知识 + 工具 + 校验”的任务；  
3. 系统可以对高风险工具生成审批并在审批后恢复执行；  
4. 至少一种异步任务（评测或报告）可以完整跑通；  
5. 可以查看报告、trace 和失败原因；  
6. 可以管理至少一套评测数据集并发起回归运行。

### 12.2 工程验收

1. 目录结构清晰；  
2. migration 可重放；  
3. API 文档可读；  
4. 错误返回一致；  
5. 关键模块具备单测和集成测试；  
6. 关键链路具备 trace。

### 12.3 安全与治理验收

1. 所有主表具备租户约束；  
2. 高风险工具无审批不能执行；  
3. 所有工具调用有审计；  
4. 日志无明文敏感字段；  
5. Prompt / Policy / Eval 可追溯。

### 12.4 演示与答辩验收

1. 可做 5~10 分钟完整演示；  
2. 可以展示系统主链路图、表结构图、观测截图、评测报告截图；  
3. 可以说明为什么不是简单 Demo；  
4. 可以量化地展示稳定性、治理能力和工程能力。

---

## 13. 测试策略

### 13.1 测试分层

#### 单元测试
覆盖：
- planner plan parser
- context assembler
- retrieval ranker
- tool schema validator
- critic rules

#### 集成测试
覆盖：
- PostgreSQL / pgvector 查询
- 文档 ingest + retrieval
- API -> DB -> model adapter 主链路
- tool registry -> policy -> audit

#### Workflow 测试
覆盖：
- 异步任务正常完成；
- 等待审批 -> 恢复；
- retry / timeout / cancel；
- versioning 兼容。

#### 合约测试
覆盖：
- API schema
- SSE event schema
- tool input/output schema
- MCP / HTTP adapter contract

#### 安全测试
覆盖：
- prompt injection
- 越权检索
- 越权工具调用
- 审批绕过
- 敏感字段泄漏

#### 评测回归测试
覆盖：
- qa dataset
- retrieval dataset
- tool_use dataset
- security dataset
- approval dataset

#### 压测 / 稳定性测试
覆盖：
- SSE 长连接；
- retrieval 热路径；
- workflow 并发；
- 批量 eval run。

### 13.2 CI 建议阶段

1. `fmt + lint`
2. `unit tests`
3. `integration tests`
4. `contract tests`
5. `eval smoke run`
6. 可选：`load smoke`

### 13.3 必须补测的变更

以下变更必须补对应测试：
- Prompt / Critic Rubric 变更 -> eval regression；
- Tool schema / policy 变更 -> contract + security test；
- 数据模型变更 -> migration + repository integration test；
- Workflow 逻辑变更 -> workflow replay / versioning test。

---

## 14. 面向汇报的价值分析

### 14.1 业务价值

1. **提升知识获取效率**：将企业制度、SOP、FAQ、案例沉淀转化为可检索、可引用、可执行的辅助能力。  
2. **缩短问题处理链路**：用户不必在知识库、工单系统、邮件/日历、表格之间来回切换。  
3. **降低专家依赖**：通过 Planner + Retrieval + Tool + Critic，把部分经验型操作标准化。  
4. **提升管理可见性**：任务、审批、失败原因、版本差异都可追踪、可汇报。

### 14.2 技术价值

1. **形成可复用 Agent 底座**：未来新增业务域时，无需重建整套链路。  
2. **沉淀评测资产**：数据集、失败案例、版本对比会成为长期技术壁垒。  
3. **建立 Agent 工程规范**：上下文工程、工具治理、审批路径、观测埋点可以复制到更多场景。  
4. **验证 Golang 在 Agent 工程化中的适配性**：证明 Go 不只是“接个模型 API”，而是能承载完整 Agent 平台。

### 14.3 组织价值

1. 为团队提供统一 Agent 开发框架；  
2. 降低每个 AI 项目重复造轮子的成本；  
3. 提供合规与审计抓手；  
4. 帮助团队建立“从 Prompt Demo 到应用系统”的共同认知。

### 14.4 汇报表达建议

向管理层汇报时，重点说：
- 效率提升；
- 风险可控；
- 可扩展；
- 可复用。

向架构评审汇报时，重点说：
- 模块边界；
- 上下文治理；
- 观测与评测；
- 审批与审计。

向面试官/答辩老师汇报时，重点说：
- 这不是聊天壳；
- 这是完整的 Agent 应用系统；
- 有系统设计、有工程治理、有质量闭环。

---

## 15. 详细风险管理表

| 风险ID | 类别 | 风险描述 | 触发信号 | 影响 | 预防措施 | 应急预案 | 责任模块 |
|---|---|---|---|---|---|---|---|
| R01 | 范围 | 一开始功能铺得太大，导致无法收敛 | backlog 持续膨胀 | 项目延期 | 冻结 V1 范围、里程碑制 | 砍掉后台与低价值工具 | PM / 全局 |
| R02 | 架构 | 过早微服务化 | 出现多个空服务仓库 | 开发效率下降 | 模块化单体优先 | 收口回单体目录 | 架构 |
| R03 | 数据 | 数据模型前期不清，后期频繁改表 | migration 反复回滚 | 代码不稳定 | 先冻结核心表 | 只允许 additive migration | Storage |
| R04 | 检索 | 文档切分或 embedding 不合理导致召回差 | 检索得分低、答非所问 | 主链路质量差 | 先做小数据集验证 | 回滚分块策略并重建索引 | Retrieval |
| R05 | 上下文 | 全量历史注入导致成本高、效果差 | token 暴涨、延迟上升 | 用户体验差 | Context Engine 分层预算 | 降级为摘要 + recent turns | Context |
| R06 | 规划 | Planner 计划不稳定 | 同样请求走不同路径 | 不可预测 | Typed plan + critic 复核 | 引入规则路由兜底 | Planner |
| R07 | 工具 | 模型滥用工具或越权调用 | 工具调用异常增多 | 安全事故 | allowlist + schema + policy | 全局关闭高风险工具 | Tool |
| R08 | 审批 | 审批流程卡死 | waiting_approval 长时间堆积 | 任务阻塞 | 审批 SLA + 过期策略 | 超时取消 / 转人工 | Workflow |
| R09 | 工作流 | Temporal 逻辑升级导致老流程重放失败 | workflow nondeterminism | 历史任务异常 | 版本策略、可回放测试 | 冻结旧 worker，灰度新版本 | Workflow |
| R10 | 观测 | trace 丢失导致无法定位问题 | 无法关联 task 与 span | 排障困难 | 统一 trace 注入 | 回退到 task_steps 审计排查 | Observability |
| R11 | 评测 | 没有评测集，改动后质量退化 | 主观体验变差 | 无法证明进步 | 从第一阶段开始建 dataset | 人工挑选坏案例补集 | Eval |
| R12 | 安全 | prompt injection 绕过工具策略 | 出现非预期外部调用 | 严重安全风险 | policy 不依赖 prompt | 熔断工具层并审计追查 | Security |
| R13 | 租户 | 跨租户数据泄漏 | 检索到其他租户数据 | 合规事故 | 所有表强制 tenant_id | 立即封禁查询并审计导出 | Auth/Storage |
| R14 | 成本 | 模型/检索成本失控 | token / API 费用上升 | 难以持续 | 摘要压缩、缓存、预算控制 | 降级模型 / 降级工具 | Runtime |
| R15 | 外部依赖 | 第三方工具或模型服务不稳定 | timeout/5xx 增加 | 链路失败 | timeout + retry + circuit breaker | 走降级回答或转异步 | Tool/Model |
| R16 | 交付 | 后端还没稳就先做 UI | 前端频繁返工 | 人力浪费 | 后端契约优先 | UI 仅做最小版 | Web |
| R17 | 演示 | 演示环境不稳定 | 演示时超时/连不上 | 影响答辩 | staging 彩排、准备回放数据 | 切换录屏/本地 mock demo | 全局 |
| R18 | 文档 | 代码有了但文档缺失 | 新人无法接手 | 不可维护 | 每个里程碑更新文档 | 发版前文档 gate | Docs |
| R19 | 发布 | 版本上线前无门禁 | 上线后质量暴跌 | 信任受损 | 评测 + smoke gate | 快速回滚到上个版本 | Eval/Release |
| R20 | 法务/合规 | 审计与导出控制不足 | 敏感数据流失 | 高风险 | 默认脱敏、审批导出 | 关停导出能力并复盘 | Security |

---

## 16. 很细的部署拓扑

### 16.1 环境分层

- **Local**：本地开发，Docker Compose，单机组件。  
- **Staging**：演示与联调环境，尽量接近生产。  
- **Production-like**：答辩/展示/未来扩展用，强调稳定性与可恢复。

### 16.2 逻辑拓扑

```text
                        ┌────────────────────────────┐
                        │        Browser/Admin       │
                        └──────────────┬─────────────┘
                                       │ HTTPS
                              ┌────────v────────┐
                              │ Ingress / LB    │
                              └────────┬────────┘
                                       │
                     ┌─────────────────┴─────────────────┐
                     │                                   │
              ┌──────v──────┐                     ┌──────v──────┐
              │ API Pods    │ <--- SSE ---------- │ Admin Web   │
              └──────┬──────┘                     └─────────────┘
                     │
         ┌───────────┼───────────┬───────────┬─────────────┐
         │           │           │           │             │
 ┌───────v───────┐ ┌─v────────┐ ┌v─────────┐ ┌v─────────┐ ┌v────────────┐
 │ PostgreSQL    │ │ Redis    │ │ Temporal │ │ OTel     │ │ ObjectStore │
 │ + pgvector    │ │          │ │ Service  │ │ Collector │ │ S3 / MinIO  │
 └───────┬───────┘ └──────────┘ └────┬──────┘ └────┬──────┘ └────────────┘
         │                           │              │
         │                           │              v
         │                           │        ┌──────────────┐
         │                           │        │ Langfuse /   │
         │                           │        │ Trace Backend│
         │                           │        └──────────────┘
         │                           │
         │                   ┌───────v────────┐
         │                   │ Worker Pods     │
         │                   └───────┬────────┘
         │                           │
         │                 ┌─────────┴──────────┐
         │                 │                    │
   ┌─────v─────┐     ┌─────v─────┐       ┌─────v──────────┐
   │ Model API │     │ MCP Tools │       │ Internal Tools │
   └───────────┘     └───────────┘       └────────────────┘
```

### 16.3 网络与安全边界

1. **公网层**：只暴露 Ingress / LB。  
2. **应用层**：API、Admin、Worker 在私有网络。  
3. **数据层**：PostgreSQL、Redis、Object Store 只允许应用层访问。  
4. **运维层**：管理后台和审批后台建议 VPN / Zero Trust 访问。  
5. **第三方依赖层**：模型 API、MCP 外部工具通过出口网关访问。

### 16.4 组件建议

#### API
- 2~3 个副本；
- 无状态；
- 滚动更新；
- SSE 需要合理的连接超时和 keepalive。

#### Worker
- 2 个以上副本；
- 支持异步任务并发；
- 分离长任务队列与短任务队列。

#### PostgreSQL
- 主从或托管实例；
- 定时备份；
- 关键索引监控；
- pgvector 与业务表共库但逻辑分域。

#### Redis
- 用于缓存、幂等键、短期协调，不作为长期事实存储。

#### Temporal
- 优先用托管/云服务或稳定自建集群；
- 所有长任务、审批等待、批量评测都走此层。

#### OTel Collector
- 独立部署；
- 负责接收应用 traces/metrics，并转发到后端。

### 16.5 部署最小建议

#### Local
- Docker Compose：Postgres、Redis、Temporal、OTel Collector、MinIO。
- API / Worker 本地进程运行。

#### Staging
- Kubernetes 或轻量云主机编排；
- 单区，多副本 API/Worker；
- 托管 Postgres / Redis 更优。

#### Production-like
- K8s namespace 隔离：`opspilot-api`, `opspilot-worker`, `opspilot-admin`；
- Secret 管理：Vault / External Secrets；
- 发布方式：蓝绿或滚动；
- 备份与恢复脚本纳入 runbook。

---

## 17. 完整 Runbook

### 17.1 本地开发启动

1. 配置 `.env.local`；  
2. 启动依赖：Postgres / Redis / Temporal / OTel / MinIO；  
3. 执行 migration；  
4. 启动 API；  
5. 启动 Worker；  
6. 访问 `/healthz` 与 `/readyz`；  
7. 跑 smoke test。

### 17.2 本地启动检查清单

- DB 可连接；
- migration 已执行；
- worker 已订阅任务；
- trace 可见；
- sample session 可创建；
- sample document 可 ingest；
- sample eval run 可排队。

### 17.3 文档导入流程

1. 上传文档；  
2. 状态 `uploaded`；  
3. 触发 ingest；  
4. 切分与 embedding；  
5. 构建索引；  
6. 状态切换为 `indexed`；  
7. 检索 smoke case 校验。

### 17.4 发版流程

1. 合并到主分支前跑完 CI；  
2. 执行 eval smoke run；  
3. 检查安全与审批路径；  
4. 发布到 staging；  
5. 回归关键场景；  
6. 再发布 production-like；  
7. 观察 30 分钟关键指标。

### 17.5 回滚流程

触发条件：
- 关键评测集显著退化；
- 工具调用失败率异常；
- 跨租户风险；
- 工作流大面积异常。

回滚步骤：
1. 停止新流量进入高风险能力；  
2. 切回上个稳定镜像 / 配置版本；  
3. 若涉及 Prompt / Policy 变更，则回滚到上个 active version；  
4. 检查迁移是否需要人工处理；  
5. 产出事故记录。

### 17.6 审批积压处理

现象：`waiting_approval` 长时间堆积。  
处理：
1. 查看审批队列与 SLA；  
2. 判断是否是通知失败还是审批人缺失；  
3. 对超时审批做取消或转派；  
4. 对可降级任务直接返回“待人工处理”。

### 17.7 Tool 服务故障处理

现象：外部工具 5xx / timeout。  
处理：
1. 打开熔断；  
2. 重试有限次数；  
3. 若为非关键工具，降级为“仅基于知识回答”；  
4. 若为关键工具，任务转异步并告知等待；  
5. 写入审计与报告。

### 17.8 模型服务异常处理

现象：模型超时、429、质量骤降。  
处理：
1. 检查配额与错误率；  
2. 降低并发 / 缩小上下文；  
3. 必要时切换备用 provider adapter；  
4. 将高成本任务转异步；  
5. 对外提示服务降级。

### 17.9 Workflow 卡死处理

现象：任务长时间 `running` 不推进。  
处理：
1. 查询 workflow history；  
2. 判断是 activity 卡住、审批等待还是外部依赖异常；  
3. 可重试 activity 则 retry；  
4. 不可恢复则 cancel 并生成失败报告；  
5. 如涉及版本问题，保留老 worker 并用版本策略迁移。

### 17.10 索引重建流程

适用场景：切分策略变更、embedding 模型变更、召回质量下降。  
流程：
1. 新建文档版本；  
2. 异步重跑 ingest；  
3. 对比旧索引与新索引回归分数；  
4. 通过后再切换 active version；  
5. 保留旧版本一段时间。

### 17.11 Secret 轮换

1. 新旧密钥并存；  
2. 先更新依赖服务；  
3. 滚动更新 API / Worker；  
4. 验证调用成功；  
5. 清理旧密钥；  
6. 审计轮换事件。

### 17.12 数据备份与恢复

备份：
- PostgreSQL 全量 + 增量；
- 对象存储备份；
- Prompt / Eval 数据在 Git 中保存。

恢复：
1. 选择恢复点；  
2. 恢复 DB；  
3. 校验 tenant 数据完整性；  
4. 恢复对象存储；  
5. 跑 smoke retrieval 和 task smoke；  
6. 记录恢复报告。

### 17.13 事故分级建议

- **SEV1**：跨租户泄漏、审批绕过、全站不可用  
- **SEV2**：主链路失败率显著上升、批量任务卡死  
- **SEV3**：个别工具异常、个别数据集失败  
- **SEV4**：UI 问题、非关键报表错误

---

## 18. 面向简历 / 答辩的包装文案

### 18.1 项目一句话定位

**OpsPilot-Go：一个面向企业知识、工单协同与评测治理的 Golang Multi-Agent 平台。**

### 18.2 简历版项目描述（推荐）

#### 版本 A：偏工程架构
- 设计并实现基于 Golang 的企业级 Multi-Agent 平台，构建 Planner / Retrieval / Tool / Critic 四类 Agent，支持复杂任务拆解、知识检索、工具执行与结果校验。  
- 建立上下文工程体系，将会话短记忆、任务 scratchpad、用户画像与长期检索结果分层管理，并通过摘要压缩与 token budget 机制提升多轮任务稳定性。  
- 基于 Workflow 构建异步评测、报告生成与审批恢复链路，支持同步问答与长任务执行并存。  
- 搭建 AgentOps 闭环，实现 LLM 调用、检索、工具、审批、评测与版本对比的全链路追踪与自动化回归。

#### 版本 B：偏企业治理
- 搭建具备多租户隔离、工具审批、操作审计、Prompt/Eval 版本化的 Agent 平台，解决传统 AI Demo 难以落地、难以治理的问题。  
- 将知识检索、工单查询与分析报告生成整合为统一执行链路，提升复杂请求处理效率并增强结果可解释性。  
- 建立评测数据集与失败案例回流机制，支持版本对比、质量门禁与问题定位。

### 18.3 答辩 / 面试开场白（30 秒）

“这个项目不是一个单纯的聊天助手，而是一个完整的企业级 Agent 应用系统。我用 Golang 搭建了 API、会话、上下文引擎、Multi-Agent Runtime、Workflow、审批、审计、评测与观测体系，目标是让 Agent 既能执行复杂任务，又能被持续监控、持续优化和持续治理。”

### 18.4 3 分钟答辩结构

1. **为什么做**：很多 Agent 项目只是 Prompt Demo，缺乏工程系统与治理能力。  
2. **做了什么**：我做的是一个企业级 Multi-Agent 平台，不只是对话，还包含检索、工具、审批、评测和报告。  
3. **怎么做的**：主链路由 API -> Context -> Planner -> Retrieval/Tool -> Critic -> Response/Workflow 构成。  
4. **难点是什么**：上下文工程、工具权限、安全审计、版本评测。  
5. **价值是什么**：能落地、可扩展、可评估、可复用。

### 18.5 面试官常问问题的答法

**问：为什么不是单 Agent？**  
答：因为复杂企业任务天然包含规划、检索、工具执行和质量校验四种不同职责，拆开后更稳定、更可控、更易扩展。

**问：为什么强调 Context Engine？**  
答：因为上下文质量直接决定结果质量。真实场景下不能只堆聊天历史，必须做分层、预算、压缩和可追踪。

**问：为什么要做评测？**  
答：没有评测就只能靠主观体验调系统，无法证明质量提升，也无法做版本门禁。

**问：为什么用 Go？**  
答：因为项目重点不只是模型调用，而是高并发 API、工作流、审计、数据库与工程治理，Go 很适合作为底座。

### 18.6 PPT 页面建议

1. 背景与问题定义  
2. 系统定位与目标范围  
3. 主链路架构图  
4. 模块边界与数据模型  
5. Context Engine 设计  
6. Tool 权限与审批设计  
7. Eval / Observability 闭环  
8. 演示流程与截图  
9. 风险控制与部署拓扑  
10. 项目价值与后续规划

---

## 19. 结论与执行要求

1. 本项目首先是**工程项目**，其次才是模型能力项目。  
2. V1 的核心不是做“大而全”，而是做“可跑通、可治理、可答辩”。  
3. Codex / Claude Code 的工作必须严格按里程碑推进，禁止跳过地基直接堆功能。  
4. 所有代码、接口、表结构、评测集、文档、技能说明都必须围绕本文档收敛。  
5. 若新需求会破坏本计划的边界，必须先进入 backlog，再决定是否进入 V2。

---

## 20. V2 展望（非本期交付）

- 多模型 provider 路由与成本优化；
- 更丰富的 MCP 工具生态；
- 更复杂的多租户策略与审批编排；
- 更成熟的管理后台与运营看板；
- 多业务域插件化扩展；
- 更高级的在线学习与记忆治理。
