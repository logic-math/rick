/**
 * rick web REST 客户端 —— plan/api-contract.md 方法面全集。
 *
 * - baseUrl 默认同源（vite dev 态由 dev server 代理 /api → 127.0.0.1:6137）
 * - token 从 localStorage("rick-web-token") 注入 Bearer
 * - 错误统一解包 WebError → throw ApiError{code,message,status}
 * - 401 时派发全局 "rick-web:unauthorized" 事件（ui store 置 authRequired 的信号源）
 */

import {
  ApiError,
  type AddWorkspaceRequest,
  type ArchivedSessionsPage,
  type CreateSessionRequest,
  type CustomizeResult,
  type HealthInfo,
  type JobFileContent,
  type JobSummary,
  type KnowledgeFileContent,
  type KnowledgeTree,
  type ResetResult,
  type ServerConfig,
  type SessionEntries,
  type SessionInfo,
  type SessionModelsResult,
  type SessionPrompt,
  type WorkspaceEntry,
} from "../types";

// ---- fs 浏览（多级路径选择器——目录浏览器数据源）----

/** GET /api/fs/list 响应 */
export interface FsListResult {
  path: string;
  /** 上级目录（文件系统根为空串） */
  parent: string;
  entries: { path: string; name: string }[];
}

/** GET /api/fs/status 响应 */
export interface FsStatusResult {
  path: string;
  exists: boolean;
  is_dir: boolean;
  has_rick: boolean;
}

export const TOKEN_STORAGE_KEY = "rick-web-token";
export const UNAUTHORIZED_EVENT = "rick-web:unauthorized";

export function getStoredToken(): string {
  try {
    return localStorage.getItem(TOKEN_STORAGE_KEY) ?? "";
  } catch {
    return "";
  }
}

export function setStoredToken(token: string): void {
  try {
    if (token) {
      localStorage.setItem(TOKEN_STORAGE_KEY, token);
    } else {
      localStorage.removeItem(TOKEN_STORAGE_KEY);
    }
  } catch {
    // localStorage 不可用（隐私模式等）——静默降级，请求将以未授权形态失败
  }
}

export class ApiClient {
  private readonly baseUrl: string;

  constructor(baseUrl: string = "") {
    // 去掉尾部斜杠，保证 path 拼接干净
    this.baseUrl = baseUrl.replace(/\/+$/, "");
  }

  // ----------------------------------------------------------
  // 内部：请求 + 错误解包
  // ----------------------------------------------------------

  private async request<T>(
    method: string,
    path: string,
    opts: { body?: unknown; query?: Record<string, string | undefined> } = {},
  ): Promise<T> {
    const url = new URL(this.baseUrl + path, window.location.origin);
    if (opts.query) {
      for (const [k, v] of Object.entries(opts.query)) {
        if (v !== undefined && v !== "") {
          url.searchParams.set(k, v);
        }
      }
    }

    const headers: Record<string, string> = {};
    const token = getStoredToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    if (opts.body !== undefined) {
      headers["Content-Type"] = "application/json";
    }

    let resp: Response;
    try {
      resp = await fetch(url.toString(), {
        method,
        headers,
        body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      });
    } catch (err) {
      throw new ApiError(
        "network_error",
        `请求失败（服务不可达或网络错误）：${err instanceof Error ? err.message : String(err)}`,
        0,
      );
    }

    if (resp.status === 401) {
      // 全局未授权信号——ui store 监听
      window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT));
    }

    // 204 无 body
    if (resp.status === 204) {
      return undefined as T;
    }

    let payload: unknown = null;
    const text = await resp.text();
    if (text) {
      try {
        payload = JSON.parse(text);
      } catch {
        payload = text;
      }
    }

    if (!resp.ok) {
      const body = payload as { error?: { code?: string; message?: string } } | string | null;
      let code = `http_${resp.status}`;
      let message = `请求失败（HTTP ${resp.status}）`;
      if (body && typeof body === "object" && body.error) {
        code = body.error.code ?? code;
        message = body.error.message ?? message;
      } else if (typeof payload === "string" && payload) {
        message = payload.slice(0, 500);
      }
      throw new ApiError(code, message, resp.status);
    }

    return payload as T;
  }

  // ----------------------------------------------------------
  // Server
  // ----------------------------------------------------------

  /** GET /api/config */
  getConfig(): Promise<ServerConfig> {
    return this.request<ServerConfig>("GET", "/api/config");
  }

  /** GET /api/health（无鉴权——探活） */
  getHealth(): Promise<HealthInfo> {
    return this.request<HealthInfo>("GET", "/api/health");
  }

  // ----------------------------------------------------------
  // Workspaces
  // ----------------------------------------------------------

  /** GET /api/workspaces */
  getWorkspaces(): Promise<WorkspaceEntry[]> {
    return this.request<WorkspaceEntry[]>("GET", "/api/workspaces");
  }

  /** POST /api/workspaces（幂等：已注册同 path → 200 返回既有） */
  async addWorkspace(req: AddWorkspaceRequest): Promise<WorkspaceEntry> {
    return this.request<WorkspaceEntry>("POST", "/api/workspaces", { body: req });
  }

  /** DELETE /api/workspaces/{id} → 204 */
  removeWorkspace(id: string): Promise<void> {
    return this.request<void>("DELETE", `/api/workspaces/${encodeURIComponent(id)}`);
  }

  /** PUT /api/workspaces/order {ids:[...]} → 204（侧边栏拖拽排序持久化） */
  reorderWorkspaces(ids: string[]): Promise<void> {
    return this.request<void>("PUT", "/api/workspaces/order", { body: { ids } });
  }

  // ----------------------------------------------------------
  // FS 浏览（多级路径选择器）
  // ----------------------------------------------------------

  /** GET /api/fs/list?path=<dir> —— 列出子目录（逐级导航） */
  fsList(path: string): Promise<FsListResult> {
    return this.request<FsListResult>("GET", "/api/fs/list", {
      query: { path: path || "/" },
    });
  }

  /** GET /api/fs/status?path=<dir> —— 目录状态（注册/创建判定） */
  fsStatus(path: string): Promise<FsStatusResult> {
    return this.request<FsStatusResult>("GET", "/api/fs/status", {
      query: { path: path || "/" },
    });
  }

  /** POST /api/fs/mkdir —— 在浏览器中新建子目录 */
  fsMkdir(path: string, name: string): Promise<{ path: string }> {
    return this.request<{ path: string }>("POST", "/api/fs/mkdir", {
      body: { path, name },
    });
  }

  // ----------------------------------------------------------
  // Sessions
  // ----------------------------------------------------------

  /** GET /api/sessions?workspace=<ws_id> */
  listSessions(workspaceId: string): Promise<SessionInfo[]> {
    return this.request<SessionInfo[]>("GET", "/api/sessions", {
      query: { workspace: workspaceId },
    });
  }

  /** POST /api/sessions → 201 SessionInfo */
  createSession(req: CreateSessionRequest): Promise<SessionInfo> {
    return this.request<SessionInfo>("POST", "/api/sessions", { body: req });
  }

  /** GET /api/sessions/{id} */
  getSession(id: string): Promise<SessionInfo> {
    return this.request<SessionInfo>("GET", `/api/sessions/${encodeURIComponent(id)}`);
  }

  /**
   * GET /api/sessions?workspace=<ws>&archived=true&limit&offset → 分页对象
   * （created_at desc；默认列表裸数组不含已归档——见 listSessions）
   */
  listArchivedSessions(
    workspaceId: string,
    opts: { limit?: number; offset?: number } = {},
  ): Promise<ArchivedSessionsPage> {
    return this.request<ArchivedSessionsPage>("GET", "/api/sessions", {
      query: {
        workspace: workspaceId,
        archived: "true",
        ...(opts.limit != null ? { limit: String(opts.limit) } : {}),
        ...(opts.offset != null ? { offset: String(opts.offset) } : {}),
      },
    });
  }

  /** POST /api/sessions/{id}/archive → 204（人工归档，幂等；active 会话先终止变 closed） */
  archiveSession(id: string): Promise<void> {
    return this.request<void>(
      "POST",
      `/api/sessions/${encodeURIComponent(id)}/archive`,
      { body: {} },
    );
  }

  /** POST /api/sessions/{id}/unarchive → 204（恢复出列表，幂等；不改 status） */
  unarchiveSession(id: string): Promise<void> {
    return this.request<void>(
      "POST",
      `/api/sessions/${encodeURIComponent(id)}/unarchive`,
      { body: {} },
    );
  }

  /** POST /api/sessions/{id}/prompt {message} → 202 */
  prompt(id: string, message: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/prompt`, {
      body: { message },
    });
  }

  /** POST /api/sessions/{id}/steer {message} → 202 */
  steer(id: string, message: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/steer`, {
      body: { message },
    });
  }

  /** POST /api/sessions/{id}/abort → 202 */
  abort(id: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/abort`, {
      body: {},
    });
  }

  /** POST /api/sessions/{id}/close → 202（幂等：已 closed 仍 202） */
  closeSession(id: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/close`, {
      body: {},
    });
  }

  /** POST /api/sessions/{id}/resume → 202（closed→active） */
  /** POST /api/sessions/import {workspace_id, job} → SessionInfo
   *  导入 CLI 启动的 job 为 web 会话（读 doing/session_id 里的 pi 会话）并立即恢复。 */
  importSession(workspaceId: string, jobId: string): Promise<SessionInfo> {
    return this.request<SessionInfo>("POST", "/api/sessions/import", {
      body: { workspace_id: workspaceId, job: jobId },
    });
  }

  resumeSession(id: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/resume`, {
      body: {},
    });
  }

  /** POST /api/sessions/{id}/ui_response {request_id, value?|confirmed?|cancelled} → 202 */
  uiResponse(
    id: string,
    req: {
      request_id: string;
      value?: string;
      confirmed?: boolean;
      cancelled?: boolean;
    },
  ): Promise<void> {
    return this.request<void>(
      "POST",
      `/api/sessions/${encodeURIComponent(id)}/ui_response`,
      { body: req },
    );
  }

  /** GET /api/sessions/{id}/entries?since=<entryId>&limit=N&before=<entryId>（active=rpc 透传；closed=离线 JSONL；before+limit 用于历史分页） */
  getEntries(
    id: string,
    opts?: { since?: string; limit?: number; before?: string },
  ): Promise<SessionEntries> {
    return this.request<SessionEntries>("GET", `/api/sessions/${encodeURIComponent(id)}/entries`, {
      query: {
        since: opts?.since,
        ...(opts?.limit != null ? { limit: String(opts.limit) } : {}),
        ...(opts?.before ? { before: opts.before } : {}),
      },
    });
  }

  /** GET /api/workspaces/{ws}/sessions/{id}/prompt —— 会话的系统提示词原文（method+instance 两份全文） */
  getSessionPrompt(workspaceId: string, sessionId: string): Promise<SessionPrompt> {
    return this.request<SessionPrompt>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(sessionId)}/prompt`,
    );
  }

  /** GET /api/sessions/{id}/models —— 可用模型列表（active 且有 worker → 200；否则 409） */
  getSessionModels(id: string): Promise<SessionModelsResult> {
    return this.request<SessionModelsResult>(
      "GET",
      `/api/sessions/${encodeURIComponent(id)}/models`,
    );
  }

  /** POST /api/sessions/{id}/model {provider, model_id} → 202 */
  setSessionModel(id: string, provider: string, modelId: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/model`, {
      body: { provider, model_id: modelId },
    });
  }

  /** POST /api/sessions/{id}/thinking {level} → 202 */
  setSessionThinking(id: string, level: string): Promise<void> {
    return this.request<void>("POST", `/api/sessions/${encodeURIComponent(id)}/thinking`, {
      body: { level },
    });
  }

  // ----------------------------------------------------------
  // Jobs（只读）
  // ----------------------------------------------------------

  /** GET /api/workspaces/{ws}/jobs —— includeArchived=true 时返回全部（归档的带 archived:true） */
  listJobs(workspaceId: string, includeArchived?: boolean): Promise<JobSummary[]> {
    return this.request<JobSummary[]>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/jobs`,
      includeArchived ? { query: { include_archived: "true" } } : undefined,
    );
  }

  /** POST /api/workspaces/{ws}/jobs/{job}/archive → 204（仅已完成 job 可归档，幂等） */
  /** POST …/jobs/{job}/archive → 204。opts.force=true = 「关闭」：允许归档未完成的
   *  job（历史 blocked/error 卡住的 job 否则永远无法归档、永久留在列表）。 */
  archiveJob(workspaceId: string, jobId: string, opts?: { force?: boolean }): Promise<void> {
    return this.request<void>(
      "POST",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/jobs/${encodeURIComponent(jobId)}/archive${
        opts?.force ? "?force=true" : ""
      }`,
    );
  }

  /** POST /api/workspaces/{ws}/jobs/{job}/unarchive → 204（恢复，幂等） */
  unarchiveJob(workspaceId: string, jobId: string): Promise<void> {
    return this.request<void>(
      "POST",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/jobs/${encodeURIComponent(jobId)}/unarchive`,
    );
  }

  /** GET /api/workspaces/{ws}/jobs/{job}/tasks → tasks.json 原文（JSON） */
  getTasks(workspaceId: string, jobId: string): Promise<unknown> {
    return this.request<unknown>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/jobs/${encodeURIComponent(jobId)}/tasks`,
    );
  }

  /** GET /api/workspaces/{ws}/jobs/{job}/file?path=plan/task1.md */
  getJobFile(workspaceId: string, jobId: string, path: string): Promise<JobFileContent> {
    return this.request<JobFileContent>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/jobs/${encodeURIComponent(jobId)}/file`,
      { query: { path } },
    );
  }

  // ----------------------------------------------------------
  // Knowledge（只读）
  // ----------------------------------------------------------

  /** GET /api/workspaces/{ws}/knowledge/tree */
  knowledgeTree(workspaceId: string): Promise<KnowledgeTree> {
    return this.request<KnowledgeTree>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/knowledge/tree`,
    );
  }

  /** GET /api/workspaces/{ws}/knowledge/file?path=domain/bugs.md */
  knowledgeFile(workspaceId: string, path: string): Promise<KnowledgeFileContent> {
    return this.request<KnowledgeFileContent>(
      "GET",
      `/api/workspaces/${encodeURIComponent(workspaceId)}/knowledge/file`,
      { query: { path } },
    );
  }

  // ----------------------------------------------------------
  // Web 管理（自迭代）
  // ----------------------------------------------------------

  /** POST /api/web/customize → {ok, scaffolded} */
  customize(): Promise<CustomizeResult> {
    return this.request<CustomizeResult>("POST", "/api/web/customize", { body: {} });
  }

  /** POST /api/web/reset → {ok} */
  reset(): Promise<ResetResult> {
    return this.request<ResetResult>("POST", "/api/web/reset", { body: {} });
  }
}

/** 全局单例（页面/组件直接 import 使用；测试可 new ApiClient()） */
export const api = new ApiClient();
