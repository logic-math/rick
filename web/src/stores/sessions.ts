/**
 * 会话列表状态：按 workspace 分组的会话表 + 状态同步。
 *
 * 数据源：
 * - REST：listSessions(workspaceId) 快照
 * - SSE：session_state 事件 → 本地状态同步（active/running/closed/error）
 * - SSE resync（重连）→ 重新拉取可见工作区的快照
 */

import { create } from "zustand";
import { api } from "../api/client";
import { sse, SSE_RESYNC_EVENT } from "../api/sse";
import type { SessionInfo } from "../types";

export interface SessionsState {
  /** workspace_id → 该工作区的会话列表 */
  byWorkspace: Map<string, SessionInfo[]>;
  /** 各工作区加载态 */
  loadingByWorkspace: Map<string, boolean>;
  /** 会话 id → SessionInfo（跨工作区索引，便于路由直达 /session/:id） */
  index: Map<string, SessionInfo>;

  /** 拉取某工作区会话列表 */
  load: (workspaceId: string) => Promise<void>;
  /** 创建会话（成功后追加进对应工作区分组） */
  create: (req: Parameters<typeof api.createSession>[0]) => Promise<SessionInfo>;
  /** 按 id 取会话（优先 index 缓存；miss 时回源） */
  get: (id: string) => Promise<SessionInfo | null>;
  /** SSE session_state 同步（本地即时更新） */
  applyState: (sessionId: string, status: SessionInfo["status"], reason?: string) => void;
  /** 从列表移除（归档事件驱动：默认列表不含已归档会话——多标签页/其他入口归档时
   *  本页也要立刻隐藏，而不依赖当前页自己触发的那次 reload）。 */
  drop: (sessionId: string) => void;
  /** 重新拉取全部已加载工作区（SSE resync 后） */
  resyncAll: () => Promise<void>;
}

function upsert(list: SessionInfo[], info: SessionInfo): SessionInfo[] {
  const i = list.findIndex((s) => s.id === info.id);
  if (i === -1) return [...list, info];
  const next = [...list];
  next[i] = info;
  return next;
}

function rebuildIndex(byWorkspace: Map<string, SessionInfo[]>): Map<string, SessionInfo> {
  const index = new Map<string, SessionInfo>();
  for (const list of byWorkspace.values()) {
    for (const s of list) index.set(s.id, s);
  }
  return index;
}

export const useSessionsStore = create<SessionsState>((set, get) => ({
  byWorkspace: new Map(),
  loadingByWorkspace: new Map(),
  index: new Map(),

  load: async (workspaceId) => {
    set((s) => ({
      loadingByWorkspace: new Map(s.loadingByWorkspace).set(workspaceId, true),
    }));
    try {
      const list = await api.listSessions(workspaceId);
      set((s) => {
        const byWorkspace = new Map(s.byWorkspace).set(workspaceId, list);
        return { byWorkspace, index: rebuildIndex(byWorkspace) };
      });
    } finally {
      set((s) => ({
        loadingByWorkspace: new Map(s.loadingByWorkspace).set(workspaceId, false),
      }));
    }
  },

  create: async (req) => {
    const info = await api.createSession(req);
    set((s) => {
      const list = s.byWorkspace.get(req.workspace_id) ?? [];
      const byWorkspace = new Map(s.byWorkspace).set(req.workspace_id, upsert(list, info));
      return { byWorkspace, index: rebuildIndex(byWorkspace) };
    });
    return info;
  },

  get: async (id) => {
    const cached = get().index.get(id);
    if (cached) return cached;
    try {
      const info = await api.getSession(id);
      set((s) => {
        const list = s.byWorkspace.get(info.workspace_id) ?? [];
        const byWorkspace = new Map(s.byWorkspace).set(info.workspace_id, upsert(list, info));
        return { byWorkspace, index: rebuildIndex(byWorkspace) };
      });
      return info;
    } catch {
      return null;
    }
  },

  drop: (sessionId) => {
    set((s) => {
      const existing = s.index.get(sessionId);
      if (!existing) return s;
      const list = (s.byWorkspace.get(existing.workspace_id) ?? []).filter((x) => x.id !== sessionId);
      const byWorkspace = new Map(s.byWorkspace).set(existing.workspace_id, list);
      return { byWorkspace, index: rebuildIndex(byWorkspace) };
    });
  },

  applyState: (sessionId, status, reason) => {
    set((s) => {
      const existing = s.index.get(sessionId);
      if (!existing) return s; // 未知会话——等 REST 快照
      const updated: SessionInfo = {
        ...existing,
        status,
        closed_at: status === "closed" || status === "error" ? new Date().toISOString() : existing.closed_at,
      };
      void reason; // reason 暂存于 events store；此处只同步状态机字段
      const list = s.byWorkspace.get(existing.workspace_id) ?? [];
      const byWorkspace = new Map(s.byWorkspace).set(existing.workspace_id, upsert(list, updated));
      return { byWorkspace, index: rebuildIndex(byWorkspace) };
    });
  },

  resyncAll: async () => {
    const ids = [...get().byWorkspace.keys()];
    await Promise.allSettled(ids.map((id) => get().load(id)));
  },
}));

// ----------------------------------------------------------
// SSE 接线（App 挂载时调用一次；幂等）
// ----------------------------------------------------------

let wired = false;

export function wireSessions(): void {
  if (wired || typeof window === "undefined") return;
  wired = true;

  sse.on("session_state", (envelope) => {
    if (!envelope.session_id) return;
    const data = envelope.data as { status?: SessionInfo["status"] };
    if (!data?.status) return;
    useSessionsStore.getState().applyState(envelope.session_id, data.status);
  });

  window.addEventListener(SSE_RESYNC_EVENT, () => {
    void useSessionsStore.getState().resyncAll();
  });
}
