/**
 * rick web 前端类型定义 —— plan/api-contract.md 的逐字段 TS 映射（单源对齐，不得臆造）。
 *
 * 契约要点：
 * - 错误统一 {"error":{"code":"<snake_case>","message":"..."}}
 * - 时间戳一律 RFC3339 带时区（string）
 * - SessionInfo.status ∈ active | running | closed | error
 */

// ============================================================
// Server / 通用
// ============================================================

/** GET /api/config 响应 */
export interface ServerConfig {
  version: number;
  rick_version: string;
  port: number;
  auth_required: boolean;
}

/** 统一错误体（HTTP 响应 body） */
export interface WebErrorBody {
  error: {
    code: string; // snake_case
    message: string; // 人读
  };
}

/** GET /api/health 响应 */
export interface HealthInfo {
  status: string; // "ok"
}

/** POST /api/web/customize 响应 */
export interface CustomizeResult {
  ok: boolean;
  /** false = 覆盖层已存在，本次跳过 */
  scaffolded: boolean;
}

/** POST /api/web/reset 响应 */
export interface ResetResult {
  ok: boolean;
}

// ============================================================
// Workspaces
// ============================================================

export interface WorkspaceEntry {
  id: string; // sha1(path) 前 8 位
  path: string; // /abs/path
  name: string;
  added_at: string; // RFC3339
  /** GET 列表时由 routes 层 join 补充；读取失败计 -1 */
  jobs_count?: number;
}

/** POST /api/workspaces 请求体 */
export interface AddWorkspaceRequest {
  path: string;
  name?: string;
}

// ============================================================
// Sessions
// ============================================================

export type SessionType =
  | "plan"
  | "easy"
  | "ctrl"
  | "human-loop"
  | "learning"
  | "dream"
  | "doing"
  /** RSI 自进化：后端把 .rick/loops/rick-rsi-loop.md 全文注入为 method 系统提示词；
   *  workspace 必须是 rick 源码的 **dev 工作区**（后端硬校验，见 task24）。 */
  | "rsi";

/**
 * 会话状态（server 权威）。五态语义：
 * - active/running：worker 在跑
 * - closed：正常结束（用户 close 或任务完成）
 * - error：执行失败（spawn 失败/doing 任务失败）
 * - suspended：**因平台升级/服务重启挂起** —— 进程不在但状态完整，可一键恢复；
 *   与 error 明确区分（人类裁决：重启后不自动恢复、不自动续跑，等人工确认）
 */
export type SessionStatus = "active" | "running" | "closed" | "error" | "suspended";

/** dream 的双模 */
export type DreamMode = "interactive" | "background";

export interface PlanParams {
  requirement: string;
  /** 复用已有 plan 目录（如 "job_5"） */
  job?: string;
}

export interface EasyParams {
  requirement: string;
  ctx_path?: string;
}

export interface CtrlParams {
  job: string;
}

export interface HumanLoopParams {
  topic: string;
}

export interface LearningParams {
  job: string;
}

export interface DreamParams {
  job_num: number; // 默认 5
  mode: DreamMode; // 默认 background
}

export interface DoingParams {
  job: string;
}

/** rsi（RSI 自进化）：无表单参数——loop 由后端按 workspace 解析并注入（task24） */
export type RsiParams = Record<string, never>;

export type SessionParams =
  | PlanParams
  | EasyParams
  | CtrlParams
  | HumanLoopParams
  | LearningParams
  | DreamParams
  | DoingParams
  | RsiParams;

/** POST /api/sessions 请求体 */
export interface CreateSessionRequest {
  workspace_id: string;
  type: SessionType;
  params: SessionParams;
  title?: string;
}

export interface SessionInfo {
  id: string; // uuid
  workspace_id: string;
  type: SessionType;
  params: Record<string, unknown>;
  title?: string;
  status: SessionStatus;
  pi_session_id: string; // uuid
  created_at: string; // RFC3339
  closed_at?: string; // RFC3339
  /** 人工归档（默认列表不含已归档会话；GET archived=true 分页返回） */
  archived?: boolean;
  archived_at?: string; // RFC3339
  /** 服务端权威的流式状态（agent 本回合是否在跑）——输入区「发送 vs 终止/steer」
   *  据此渲染；刷新/重连后不依赖客户端事件重放推断。 */
  busy?: boolean;
  /** 用户给所属 job 起的任务名（「任务名」别名，展示层；未命名为空）。
   *  侧栏会话行优先显示它——「doing job_3」看不出在干什么。 */
  job_name?: string;
  /** 后台进度日志（doing/dream；仅单会话查询返回）——监控页首次/事后打开时回填
   *  事件流（这些事件原本只在 hub 环形缓冲里活过一次）。 */
  progress?: Array<{ at: string; kind: string; text: string }>;
  /** 最近一次状态跃迁原因（服务端持久化；用于解释「为何挂起/中断」）。
   *  后端当前列表/单会话投影未暴露该字段时为空——UI 用通用文案兜底。 */
  last_reason?: string;
}

/** 恢复报告里的一行（谁被挂起 / 谁已恢复 / 谁恢复失败） */
export interface RecoveryItem {
  id: string;
  type?: string;
  title?: string;
  job_id?: string;
  reason?: string;
  /** RFC3339 */
  at: string;
}

/** GET /api/recovery —— 平台升级/重启后的挂起与恢复台账 */
export interface RecoveryReport {
  version: number;
  /** RFC3339（报告生成时刻） */
  at: string;
  suspended: RecoveryItem[];
  recovered: RecoveryItem[];
  failed: RecoveryItem[];
}

/** POST /api/sessions/{id}/continue 响应（202 幂等；doing/dream 返回归一化明细） */
export interface ContinueSessionResult {
  ok: boolean;
  already_active?: boolean;
  resumed?: boolean;
  /** doing/dream 的会话类型 */
  kind?: SessionType;
  job?: string;
  /** doing 续跑时被归一化的遗留 running task（running → pending） */
  normalized_tasks?: string[];
}

/** GET /api/sessions?workspace=..&archived=true 分页响应（区别于裸数组的默认列表） */
export interface ArchivedSessionsPage {
  items: SessionInfo[];
  total: number;
  limit: number;
  offset: number;
}

// ============================================================
// Sessions / entries（离线回放）
// ============================================================

/** GET /api/sessions/{id}/entries 响应 */
export interface SessionEntries {
  entries: SessionEntry[];
  leaf_id: string | null;
}

/** GET /api/workspaces/{ws}/sessions/{id}/prompt 响应——系统提示词全文 */
export interface SessionPrompt {
  /** method 层系统提示词（--append-system-prompt 注入的方法描述） */
  method: string;
  /** instance 实例 prompt 全文（plan_prompt.md 等） */
  instance: string;
}

/**
 * pi session JSONL 行（session-format v3 轻量视图）。
 * message 内容按 pi 原始 schema（role/content…），前端按 role 渲染。
 */
export interface SessionEntry {
  type: string; // "message" 等
  id: string;
  parentId?: string;
  timestamp?: string;
  message?: {
    role: string;
    content?: string | Array<{ type: string; text?: string }>;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

// ============================================================
// Jobs
// ============================================================

export interface TaskBrief {
  task_id: string;
  name: string;
  status: string; // pending / running / success / error
  commit_hash?: string;
}

export interface JobSummary {
  job_id: string; // "job_5"
  updated_at: string; // RFC3339
  tasks: TaskBrief[];
  /** 已归档（include_archived=true 时返回） */
  archived?: boolean;
  /** 归档来源：manual=手动归档 / dream=已被 dream 学习 / done=已完成自动归档 */
  archived_by?: "manual" | "dream" | "done";
  /** 进度阶段：planned=plan 已产出（task*.md）但未执行 doing；doing=已有 tasks.json；
   *  started=CLI 已启动（doing/session_id 存在）但 tasks.json 未写（会话结束时才写）。
   *  planned/started 的 job 也必须能被执行 doing/ctrl 的会话表单选到。 */
  stage?: "planned" | "doing" | "started";
  /** 用户自定义任务名（展示层别名；未命名为空 → UI 显示 job_id） */
  name?: string;
}

/** GET /api/workspaces/{ws}/jobs/{job}/file 响应 */
export interface JobFileContent {
  path: string;
  content: string;
}

// ============================================================
// Knowledge
// ============================================================

export interface KnowledgeFileNode {
  path: string; // "domain/bugs.md"
  size: number;
}

/** GET /api/workspaces/{ws}/knowledge/tree 响应 */
export interface KnowledgeTree {
  tree: KnowledgeFileNode[];
}

/** GET /api/workspaces/{ws}/knowledge/file 响应 */
export interface KnowledgeFileContent {
  path: string;
  content: string;
}

// ============================================================
// SSE 事件流
// ============================================================

export type SSEEventType =
  | "server_info"
  | "session_event"
  | "session_state"
  | "jobs_update"
  | "frontend_reload";

/** SSE envelope（data: 行的 JSON） */
export interface SSEEnvelope<T = unknown> {
  seq: number; // 全局单调（服务端重启归零）
  type: SSEEventType;
  session_id: string | null;
  data: T;
}

/** server_info 事件 data */
export interface ServerInfoData {
  version: number;
  rick_version: string;
  time: string;
}

/** session_event 的 data：{"event": <pi 原始 rpc 事件对象>} */
export interface SessionEventData {
  event: PiRpcEvent;
}

/**
 * pi rpc 事件透传对象（宽类型——pi 的 rpc 事件面见
 * ~/.rick/pi/agent/runtime/.../docs/rpc.md Event Types 节；前端按需窄化）。
 */
export interface PiRpcEvent {
  type: string;
  id?: string;
  sessionId?: string;
  toolCallId?: string;
  toolName?: string;
  args?: unknown;
  result?: unknown;
  isError?: boolean;
  message?: {
    role?: string;
    content?: string | Array<{ type: string; text?: string }>;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

/** session_state 事件 data */
export interface SessionStateData {
  status: SessionStatus;
  reason?: string;
}

/** jobs_update 事件 data */
export interface JobsUpdateData {
  job_id: string;
  diff: Array<{
    task_id: string;
    from: string;
    to: string;
  }>;
  snapshot: TaskBrief[];
}

/** frontend_reload 事件 data：{} */
export interface FrontendReloadData {
  [key: string]: never;
}

// ============================================================
// API 客户端错误
// ============================================================

/** ApiClient 抛出的统一错误 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

/** 会话模型（GET /api/sessions/{id}/models 的模型项） */
export interface SessionModel {
  id: string;
  name: string;
  provider: string;
}

/** GET /api/sessions/{id}/models 响应 */
export interface SessionModelsResult {
  models: SessionModel[];
  current: SessionModel | null;
}
